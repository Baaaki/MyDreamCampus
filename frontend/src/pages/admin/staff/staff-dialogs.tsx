import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AlertCircle, Loader2 } from "lucide-react"
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
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
import { useFaculties } from "@/lib/services/catalog-service"
import {
  staffService,
  toStaffUpdate,
  type StaffUpdate,
} from "@/lib/services/staff-service"
import type { Staff } from "@/lib/types"

const selectClass =
  "flex h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm disabled:cursor-not-allowed disabled:opacity-50"

function ErrorLine({ message }: { message: string }) {
  return (
    <p className="flex items-center gap-2 text-sm text-destructive">
      <AlertCircle className="h-4 w-4 shrink-0" />
      {message}
    </p>
  )
}

interface StaffEditDialogProps {
  staff: Staff | null
  onOpenChange: (open: boolean) => void
}

/**
 * Edits the department, phone and office of a staff account; open while
 * `staff` is set. Name and e-mail are shown but fixed: the backend does not
 * change them.
 */
export function StaffEditDialog({ staff, onOpenChange }: StaffEditDialogProps) {
  return (
    <Dialog open={!!staff} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        {/* Keyed so each opened account starts from its own values. */}
        {staff && (
          <StaffEditForm
            key={staff.id}
            staff={staff}
            onDone={() => onOpenChange(false)}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

function StaffEditForm({
  staff,
  onDone,
}: {
  staff: Staff
  onDone: () => void
}) {
  const queryClient = useQueryClient()
  const { data: faculties = [] } = useFaculties()
  const [form, setForm] = useState<StaffUpdate>(() => toStaffUpdate(staff))
  const [faculty, setFaculty] = useState<string | null>(null)

  // The account carries no faculty; it is the one holding its department.
  const selectedFaculty =
    faculty ??
    faculties.find((f) => f.departments.some((d) => d.name === form.department))
      ?.name ??
    ""
  const departments =
    faculties.find((f) => f.name === selectedFaculty)?.departments ?? []

  const update = useMutation({
    mutationFn: (data: StaffUpdate) => staffService.update(staff.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["staff", "list"] })
      onDone()
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    update.mutate({
      department: form.department,
      phone: form.phone.trim(),
      office_location: form.office_location.trim(),
    })
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>
          Personeli Düzenle: {staff.first_name} {staff.last_name}
        </DialogTitle>
        <DialogDescription>
          {staff.email} · Ad, soyad ve e-posta değiştirilemez.
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit_faculty">Fakülte</Label>
          <select
            id="edit_faculty"
            className={selectClass}
            value={selectedFaculty}
            onChange={(e) => {
              setFaculty(e.target.value)
              setForm({ ...form, department: "" })
            }}
          >
            <option value="">Fakülte seçin</option>
            {faculties.map((f) => (
              <option key={f.id} value={f.name}>
                {f.name}
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit_department">Bölüm</Label>
          <select
            id="edit_department"
            className={selectClass}
            value={form.department}
            disabled={!selectedFaculty}
            onChange={(e) => setForm({ ...form, department: e.target.value })}
          >
            <option value="">
              {selectedFaculty ? "Bölüm seçin" : "Önce fakülte seçin"}
            </option>
            {departments.map((d) => (
              <option key={d.id} value={d.name}>
                {d.name}
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit_phone">Telefon</Label>
          <Input
            id="edit_phone"
            value={form.phone}
            onChange={(e) => setForm({ ...form, phone: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="edit_office_location">Ofis Konumu</Label>
          <Input
            id="edit_office_location"
            value={form.office_location}
            onChange={(e) =>
              setForm({ ...form, office_location: e.target.value })
            }
          />
        </div>
        {update.isError && (
          <ErrorLine
            message={apiErrorMessage(update.error, "Personel güncellenemedi")}
          />
        )}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onDone}>
            İptal
          </Button>
          <Button type="submit" disabled={update.isPending}>
            {update.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Kaydet
          </Button>
        </DialogFooter>
      </form>
    </>
  )
}

interface StaffDeleteDialogProps {
  staff: Staff | null
  onOpenChange: (open: boolean) => void
}

/**
 * Confirms deleting a staff account; open while `staff` is set. A refusal,
 * such as the 403 of a protected account, stays in the dialog.
 */
export function StaffDeleteDialog({
  staff,
  onOpenChange,
}: StaffDeleteDialogProps) {
  const queryClient = useQueryClient()
  const remove = useMutation({
    mutationFn: (id: string) => staffService.remove(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["staff", "list"] })
      onOpenChange(false)
    },
  })

  const handleOpenChange = (open: boolean) => {
    if (!open) remove.reset()
    onOpenChange(open)
  }

  return (
    <AlertDialog open={!!staff} onOpenChange={handleOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Personel silinsin mi?</AlertDialogTitle>
          <AlertDialogDescription>
            {staff?.first_name} {staff?.last_name} ({staff?.email}) hesabı
            pasifleştirilir: giriş yapamaz ve danışmanı olduğu öğrencilerin
            danışmanlığı kaldırılır.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {remove.isError && (
          <ErrorLine
            message={apiErrorMessage(remove.error, "Personel silinemedi")}
          />
        )}
        <AlertDialogFooter>
          <AlertDialogCancel>Vazgeç</AlertDialogCancel>
          {/* A plain button: AlertDialogAction would close before the answer. */}
          <Button
            variant="destructive"
            disabled={remove.isPending}
            onClick={() => staff && remove.mutate(staff.id)}
          >
            {remove.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Sil
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
