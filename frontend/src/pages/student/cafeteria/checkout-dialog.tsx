import { useState, type ReactNode } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { HTTPError } from "ky"
import { AlertCircle, Check, CreditCard, Loader2, X } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { apiErrorMessage } from "@/lib/api-error"
import {
  EMPTY_CARD_FORM,
  TEST_CARDS,
  formatCardNumber,
  formatExpiry,
  toConfirmRequest,
  validateCardForm,
  type CardForm,
  type CardFormErrors,
} from "@/lib/payment-card"
import {
  checkoutOutcome,
  createBatchReservation,
  getMyReservations,
  type CreateReservationRequest,
} from "@/lib/services/meal-service"
import { confirmPayment } from "@/lib/services/payment-service"
import type { Payment } from "@/lib/types"

export interface CheckoutItem {
  key: string
  label: string
  menuType: "normal" | "vegan"
}

interface CheckoutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  items: CheckoutItem[]
  estimatedTotal: number
  buildRequest: () => CreateReservationRequest[]
  /** Called once the reservations are paid, to clear the week's selection. */
  onPaid: () => void
}

/** The reservations one checkout created and the payment that covers them. */
interface Checkout {
  paymentId: string
  reservationIds: string[]
  fromDate: string
  toDate: string
  amount: number
  expiresAt: string
}

type Step = "summary" | "card" | "processing" | "rejected"

const POLL_INTERVAL_MS = 1500
// Meal usually confirms within seconds; past this the student is sent to
// their reservations instead of watching a spinner.
const POLL_TIMEOUT_MS = 30_000

function formatTRY(amount: number): string {
  return `${amount.toFixed(2)} ₺`
}

/**
 * Summary → card form → confirm → wait for meal → result. The state lives
 * here, outside the dialog content, so closing the dialog mid-checkout and
 * reopening it resumes the same payment instead of reserving again.
 */
export function CheckoutDialog({
  open,
  onOpenChange,
  items,
  estimatedTotal,
  buildRequest,
  onPaid,
}: CheckoutDialogProps) {
  const queryClient = useQueryClient()
  const [step, setStep] = useState<Step>("summary")
  const [checkout, setCheckout] = useState<Checkout | null>(null)
  const [form, setForm] = useState<CardForm>(EMPTY_CARD_FORM)
  const [formErrors, setFormErrors] = useState<CardFormErrors>({})
  const [error, setError] = useState<string | null>(null)
  const [payment, setPayment] = useState<Payment | null>(null)
  const [pollStartedAt, setPollStartedAt] = useState(0)

  const reserve = useMutation({
    mutationFn: () => createBatchReservation({ reservations: buildRequest() }),
    onSuccess: (res) => {
      const dates = res.reservations.map((r) => r.date).sort()
      setCheckout({
        paymentId: res.payment_id,
        reservationIds: res.reservations.map((r) => r.id),
        fromDate: dates[0],
        toDate: dates[dates.length - 1],
        amount: res.total_amount,
        expiresAt: res.expires_at,
      })
      setError(null)
      setStep("card")
    },
    onError: (err) =>
      setError(
        apiErrorMessage(
          err,
          "Rezervasyon oluşturulamadı, lütfen tekrar deneyin"
        )
      ),
  })

  const pay = useMutation({
    mutationFn: (c: Checkout) =>
      confirmPayment(c.paymentId, toConfirmRequest(form)),
    onSuccess: (result) => {
      // The card data has done its job; do not keep it around.
      setForm(EMPTY_CARD_FORM)
      setPayment(result)
      setPollStartedAt(Date.now())
      setStep("processing")
    },
    onError: (err) => {
      // 409: the payment expired or was settled elsewhere — this checkout
      // is over. Anything else (a mistyped card) can be fixed and retried.
      if (err instanceof HTTPError && err.response.status === 409) {
        setForm(EMPTY_CARD_FORM)
        setError(apiErrorMessage(err, "Bu ödeme artık tamamlanamaz"))
        setStep("rejected")
        return
      }
      setError(
        apiErrorMessage(err, "Ödeme gönderilemedi, lütfen tekrar deneyin")
      )
    },
  })

  const poll = useQuery({
    queryKey: ["checkout-reservations", checkout?.paymentId],
    queryFn: () =>
      getMyReservations({
        from_date: checkout?.fromDate,
        to_date: checkout?.toDate,
      }),
    enabled: step === "processing" && checkout !== null,
    refetchInterval: (query) => {
      const { data, dataUpdatedAt, errorUpdatedAt } = query.state
      const lastPollAt = Math.max(dataUpdatedAt, errorUpdatedAt)
      if (lastPollAt - pollStartedAt > POLL_TIMEOUT_MS) return false
      if (
        checkout &&
        data &&
        checkoutOutcome(checkout.reservationIds, data.reservations) !==
          "pending"
      ) {
        return false
      }
      return POLL_INTERVAL_MS
    },
  })

  const outcome =
    step === "processing" && checkout && poll.data
      ? checkoutOutcome(checkout.reservationIds, poll.data.reservations)
      : "pending"
  // Failed polls count too, so an unreachable meal service ends the wait.
  const timedOut =
    step === "processing" &&
    Math.max(poll.dataUpdatedAt, poll.errorUpdatedAt) - pollStartedAt >
      POLL_TIMEOUT_MS
  const declineReason =
    payment?.status === "failed"
      ? (payment.failure_reason ?? "Ödeme reddedildi")
      : null

  type Phase = "summary" | "card" | "waiting" | "paid" | "failed" | "late"
  let phase: Phase = step === "processing" ? "waiting" : "summary"
  if (step === "card") phase = "card"
  if (step === "rejected") phase = "failed"
  if (step === "processing") {
    if (outcome === "confirmed") phase = "paid"
    else if (outcome === "cancelled") phase = "failed"
    // A declined card needs no confirmation from meal to be reported.
    else if (timedOut) phase = declineReason ? "failed" : "late"
  }

  const reset = () => {
    setStep("summary")
    setCheckout(null)
    setForm(EMPTY_CARD_FORM)
    setFormErrors({})
    setError(null)
    setPayment(null)
    queryClient.invalidateQueries({ queryKey: ["my-reservations"] })
  }

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      if (phase === "paid" || phase === "late") {
        reset()
        onPaid()
      } else if (phase === "failed") {
        reset()
      } else {
        // Keep the checkout to resume it, but not the card details.
        setForm(EMPTY_CARD_FORM)
        setFormErrors({})
      }
    }
    onOpenChange(next)
  }

  const submitCard = () => {
    if (!checkout) return
    const errors = validateCardForm(form)
    setFormErrors(errors)
    setError(null)
    if (Object.keys(errors).length === 0) pay.mutate(checkout)
  }

  const updateField = (field: keyof CardForm, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    setFormErrors((prev) => ({ ...prev, [field]: undefined }))
  }

  const fillTestCard = (number: string) => {
    setForm((prev) => ({
      number,
      expiry: prev.expiry || "12/30",
      cvc: prev.cvc || "123",
      holder: prev.holder,
    }))
    setFormErrors({})
  }

  const failureMessage = error ?? declineReason ?? "Rezervasyon iptal edildi"

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        {phase === "summary" && (
          <>
            <DialogHeader>
              <DialogTitle>Ödeme Özeti</DialogTitle>
              <DialogDescription>
                Seçimlerinizi kontrol edin. Devam ettiğinizde öğünleriniz ödeme
                süresi boyunca sizin için ayrılır.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="rounded-lg bg-gray-50 p-4 dark:bg-gray-800">
                <p className="mb-2 text-sm text-gray-600 dark:text-gray-400">
                  Seçilen Yemekler:
                </p>
                <div className="space-y-2">
                  {items.map((item) => (
                    <div
                      key={item.key}
                      className="flex items-center justify-between"
                    >
                      <span className="text-gray-900 dark:text-white">
                        {item.label}
                      </span>
                      <Badge
                        className={
                          item.menuType === "vegan"
                            ? "bg-green-500"
                            : "bg-orange-500"
                        }
                      >
                        {item.menuType === "vegan" ? "Vegan" : "Normal"}
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-emerald-50 p-4 dark:bg-emerald-900/20">
                <span className="font-medium text-gray-900 dark:text-white">
                  Toplam Tutar:
                </span>
                <span className="text-xl font-bold text-emerald-600">
                  {formatTRY(estimatedTotal)}
                </span>
              </div>
              {error && <ErrorLine message={error} />}
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={reserve.isPending}
              >
                İptal
              </Button>
              <Button
                onClick={() => reserve.mutate()}
                className="bg-emerald-600 hover:bg-emerald-700"
                disabled={reserve.isPending || items.length === 0}
              >
                {reserve.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <CreditCard className="mr-2 h-4 w-4" />
                )}
                Ödemeye Geç
              </Button>
            </DialogFooter>
          </>
        )}

        {phase === "card" && checkout && (
          <>
            <DialogHeader>
              <DialogTitle>Kart Bilgileri</DialogTitle>
              <DialogDescription>
                {checkout.reservationIds.length} öğün için{" "}
                {formatTRY(checkout.amount)} ödenecek. Ödemeyi{" "}
                {new Date(checkout.expiresAt).toLocaleTimeString("tr-TR", {
                  hour: "2-digit",
                  minute: "2-digit",
                })}{" "}
                saatine kadar tamamlayın.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-2">
              <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm dark:border-amber-900 dark:bg-amber-900/20">
                <p className="font-medium text-amber-800 dark:text-amber-200">
                  Demo ödeme: gerçek para çekilmez.
                </p>
                <p className="mt-1 text-amber-700 dark:text-amber-300">
                  Test kartlarından birini kullanın (son kullanma tarihi ileri
                  bir tarih, CVC herhangi 3 hane):
                </p>
                <ul className="mt-2 space-y-1">
                  {TEST_CARDS.map((card) => (
                    <li
                      key={card.number}
                      className="flex items-center justify-between gap-2"
                    >
                      <div>
                        <p className="font-mono whitespace-nowrap text-gray-900 dark:text-gray-100">
                          {card.number}
                        </p>
                        <p className="text-xs text-gray-600 dark:text-gray-400">
                          {card.result}
                        </p>
                      </div>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => fillTestCard(card.number)}
                      >
                        Kullan
                      </Button>
                    </li>
                  ))}
                </ul>
              </div>

              <form
                className="space-y-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  submitCard()
                }}
                noValidate
              >
                <CardField
                  id="card-number"
                  label="Kart Numarası"
                  error={formErrors.number}
                >
                  <Input
                    id="card-number"
                    inputMode="numeric"
                    autoComplete="cc-number"
                    placeholder="0000 0000 0000 0000"
                    value={form.number}
                    onChange={(e) =>
                      updateField("number", formatCardNumber(e.target.value))
                    }
                  />
                </CardField>
                <div className="grid grid-cols-2 gap-3">
                  <CardField
                    id="card-expiry"
                    label="Son Kullanma"
                    error={formErrors.expiry}
                  >
                    <Input
                      id="card-expiry"
                      inputMode="numeric"
                      autoComplete="cc-exp"
                      placeholder="AA/YY"
                      value={form.expiry}
                      onChange={(e) =>
                        updateField("expiry", formatExpiry(e.target.value))
                      }
                    />
                  </CardField>
                  <CardField id="card-cvc" label="CVC" error={formErrors.cvc}>
                    <Input
                      id="card-cvc"
                      inputMode="numeric"
                      autoComplete="cc-csc"
                      placeholder="123"
                      maxLength={4}
                      value={form.cvc}
                      onChange={(e) =>
                        updateField("cvc", e.target.value.replace(/\D/g, ""))
                      }
                    />
                  </CardField>
                </div>
                <CardField
                  id="card-holder"
                  label="Kart Üzerindeki Ad"
                  error={formErrors.holder}
                >
                  <Input
                    id="card-holder"
                    autoComplete="cc-name"
                    placeholder="Ad Soyad"
                    value={form.holder}
                    onChange={(e) => updateField("holder", e.target.value)}
                  />
                </CardField>
                {error && <ErrorLine message={error} />}
                <DialogFooter>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => handleOpenChange(false)}
                    disabled={pay.isPending}
                  >
                    Kapat
                  </Button>
                  <Button
                    type="submit"
                    className="bg-emerald-600 hover:bg-emerald-700"
                    disabled={pay.isPending}
                  >
                    {pay.isPending ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <CreditCard className="mr-2 h-4 w-4" />
                    )}
                    {formatTRY(checkout.amount)} Öde
                  </Button>
                </DialogFooter>
              </form>
            </div>
          </>
        )}

        {phase === "waiting" && (
          <>
            <DialogHeader>
              <DialogTitle>Ödeme İşleniyor</DialogTitle>
              <DialogDescription>
                Rezervasyonunuzun durumu kontrol ediliyor.
              </DialogDescription>
            </DialogHeader>
            <div className="flex flex-col items-center py-8">
              <Loader2 className="h-10 w-10 animate-spin text-emerald-600" />
            </div>
          </>
        )}

        {phase === "paid" && (
          <ResultView
            tone="success"
            title="Rezervasyon Onaylandı"
            message={
              payment?.card_last4
                ? `${payment.card_brand ?? "Kart"} •••• ${payment.card_last4} ile ${formatTRY(payment.amount)} ödendi.`
                : "Ödemeniz alındı."
            }
            primary={{ label: "Tamam", onClick: () => handleOpenChange(false) }}
          />
        )}

        {phase === "late" && (
          <ResultView
            tone="success"
            title="Ödeme Alındı"
            message="Rezervasyon onayı biraz gecikti. Durumunu Rezervasyonlarım sayfasından takip edebilirsiniz."
            primary={{ label: "Tamam", onClick: () => handleOpenChange(false) }}
          />
        )}

        {phase === "failed" && (
          <ResultView
            tone="error"
            title="Ödeme Başarısız"
            message={`${failureMessage}. Öğünleriniz ayrılmadı; tekrar deneyebilirsiniz.`}
            primary={{ label: "Tekrar Dene", onClick: reset }}
            secondary={{
              label: "Kapat",
              onClick: () => handleOpenChange(false),
            }}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

function CardField({
  id,
  label,
  error,
  children,
}: {
  id: string
  label: string
  error?: string
  children: ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}

function ErrorLine({ message }: { message: string }) {
  return (
    <div className="flex items-center gap-2 text-sm text-destructive">
      <AlertCircle className="h-4 w-4 shrink-0" />
      {message}
    </div>
  )
}

function ResultView({
  tone,
  title,
  message,
  primary,
  secondary,
}: {
  tone: "success" | "error"
  title: string
  message: string
  primary: { label: string; onClick: () => void }
  secondary?: { label: string; onClick: () => void }
}) {
  return (
    <>
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
      </DialogHeader>
      <div className="flex flex-col items-center py-6">
        <div
          className={`mb-4 flex h-16 w-16 items-center justify-center rounded-full ${
            tone === "success"
              ? "bg-emerald-100 dark:bg-emerald-900/50"
              : "bg-red-100 dark:bg-red-900/50"
          }`}
        >
          {tone === "success" ? (
            <Check className="h-8 w-8 text-emerald-600" />
          ) : (
            <X className="h-8 w-8 text-red-600" />
          )}
        </div>
        <DialogDescription className="text-center">{message}</DialogDescription>
      </div>
      <DialogFooter className="flex flex-col gap-2 sm:flex-col">
        <Button onClick={primary.onClick} className="w-full">
          {primary.label}
        </Button>
        {secondary && (
          <Button
            onClick={secondary.onClick}
            variant="outline"
            className="w-full"
          >
            {secondary.label}
          </Button>
        )}
      </DialogFooter>
    </>
  )
}
