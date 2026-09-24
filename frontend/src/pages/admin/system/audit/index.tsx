import { useState } from "react"
import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { format } from "date-fns"
import { tr } from "date-fns/locale"
import {
  ScrollText,
  RefreshCw,
  Loader2,
  ChevronLeft,
  ChevronRight,
  Search,
} from "lucide-react"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

import type { AuditLogEntry } from "@/lib/types"
import { listAuditLog } from "@/lib/services/system-service"
import type { AuditLogFilters } from "@/lib/services/system-service"

const PAGE_SIZE = 20

const SERVICE_COLORS: Record<string, string> = {
  catalog: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300",
  enrollment:
    "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300",
  grades:
    "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300",
  meal: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300",
  attendance:
    "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300",
  auth: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300",
  students: "bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300",
  staff: "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300",
  payment:
    "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300",
}

export default function AuditLogPage() {
  const [page, setPage] = useState(0)

  // Filters
  const [filterService, setFilterService] = useState("")
  const [filterAction, setFilterAction] = useState("")

  // Detail modal
  const [selectedEntry, setSelectedEntry] = useState<AuditLogEntry | null>(null)

  const {
    data,
    isFetching: loading,
    isError,
    refetch: fetchLogs,
  } = useQuery({
    queryKey: ["system", "audit-log", page, filterService, filterAction],
    queryFn: () => {
      const filters: AuditLogFilters = {
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      }
      if (filterService) filters.service = filterService
      if (filterAction) filters.action = filterAction
      return listAuditLog(filters)
    },
    placeholderData: keepPreviousData,
  })
  const entries: AuditLogEntry[] = data?.data || []
  const total = data?.total || 0

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const handleSearch = () => {
    setPage(0)
    fetchLogs()
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          Audit Log
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Tum kritik islemlerin degistirilemez kayitlari (DB trigger ile
          korunur)
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <ScrollText className="h-5 w-5 text-indigo-600" />
            <CardTitle>Islem Kayitlari</CardTitle>
            <span className="ml-auto text-sm text-gray-500">{total} kayit</span>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Filters */}
          <div className="flex flex-wrap items-end gap-3">
            <div className="w-40">
              <Label>Servis</Label>
              <select
                value={filterService}
                onChange={(e) => setFilterService(e.target.value)}
                className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:outline-none"
              >
                <option value="">Tumu</option>
                <option value="catalog">Catalog</option>
                <option value="enrollment">Enrollment</option>
                <option value="grades">Grades</option>
                <option value="meal">Meal</option>
                <option value="attendance">Attendance</option>
                <option value="auth">Auth</option>
                <option value="students">Students</option>
                <option value="staff">Staff</option>
                <option value="payment">Payment</option>
              </select>
            </div>
            <div className="max-w-xs flex-1">
              <Label>Islem (action)</Label>
              <Input
                placeholder="semester.activated, period.created..."
                value={filterAction}
                onChange={(e) => setFilterAction(e.target.value)}
                className="mt-1"
              />
            </div>
            <Button variant="outline" size="sm" onClick={handleSearch}>
              <Search className="mr-1 h-4 w-4" />
              Filtrele
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchLogs()}
              disabled={loading}
            >
              <RefreshCw
                className={`h-4 w-4 ${loading ? "animate-spin" : ""}`}
              />
            </Button>
          </div>

          {/* Table */}
          <div className="rounded-lg border border-gray-200 dark:border-gray-700">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[160px]">Zaman</TableHead>
                  <TableHead className="w-[100px]">Servis</TableHead>
                  <TableHead>Islem</TableHead>
                  <TableHead>Kaynak</TableHead>
                  <TableHead>Aktor</TableHead>
                  <TableHead className="w-[80px] text-right">Detay</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={6} className="py-6 text-center">
                      <Loader2 className="mx-auto h-5 w-5 animate-spin text-gray-400" />
                    </TableCell>
                  </TableRow>
                ) : isError ? (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className="py-6 text-center text-destructive"
                    >
                      Audit log yuklenemedi
                    </TableCell>
                  </TableRow>
                ) : entries.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className="py-6 text-center text-gray-500"
                    >
                      Kayit bulunamadi
                    </TableCell>
                  </TableRow>
                ) : (
                  entries.map((entry) => (
                    <TableRow
                      key={entry.id}
                      className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50"
                      onClick={() => setSelectedEntry(entry)}
                    >
                      <TableCell className="font-mono text-xs text-gray-500">
                        {format(new Date(entry.timestamp), "dd MMM HH:mm:ss", {
                          locale: tr,
                        })}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant="secondary"
                          className={SERVICE_COLORS[entry.service] || ""}
                        >
                          {entry.service}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm font-medium">
                        {entry.action}
                      </TableCell>
                      <TableCell className="text-sm text-gray-500">
                        {entry.resource_type}
                        {entry.resource_id && (
                          <span className="ml-1 font-mono text-xs text-gray-400">
                            {entry.resource_id.length > 8
                              ? entry.resource_id.slice(0, 8) + "..."
                              : entry.resource_id}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-gray-500">
                        {entry.actor_role && (
                          <span className="mr-1 text-xs font-medium uppercase">
                            {entry.actor_role}
                          </span>
                        )}
                        {entry.actor_id ? (
                          <span className="font-mono text-xs">
                            {entry.actor_id.slice(0, 8)}...
                          </span>
                        ) : (
                          <span className="text-gray-400">system</span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation()
                            setSelectedEntry(entry)
                          }}
                        >
                          ...
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-500">
                Sayfa {page + 1} / {totalPages}
              </span>
              <div className="flex gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                  disabled={page === 0}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    setPage((p) => Math.min(totalPages - 1, p + 1))
                  }
                  disabled={page >= totalPages - 1}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Detail Modal */}
      <Dialog
        open={!!selectedEntry}
        onOpenChange={(open) => {
          if (!open) setSelectedEntry(null)
        }}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Audit Log Detayi</DialogTitle>
          </DialogHeader>
          {selectedEntry && (
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div>
                  <span className="text-gray-500">Zaman:</span>
                  <p className="font-medium">
                    {format(
                      new Date(selectedEntry.timestamp),
                      "dd MMMM yyyy HH:mm:ss",
                      { locale: tr }
                    )}
                  </p>
                </div>
                <div>
                  <span className="text-gray-500">Servis:</span>
                  <p className="font-medium">{selectedEntry.service}</p>
                </div>
                <div>
                  <span className="text-gray-500">Islem:</span>
                  <p className="font-medium">{selectedEntry.action}</p>
                </div>
                <div>
                  <span className="text-gray-500">Aktor Rolu:</span>
                  <p className="font-medium">
                    {selectedEntry.actor_role || "-"}
                  </p>
                </div>
                <div>
                  <span className="text-gray-500">Aktor ID:</span>
                  <p className="font-mono text-xs break-all">
                    {selectedEntry.actor_id || "-"}
                  </p>
                </div>
                <div>
                  <span className="text-gray-500">Kaynak Tipi:</span>
                  <p className="font-medium">
                    {selectedEntry.resource_type || "-"}
                  </p>
                </div>
                <div className="col-span-2">
                  <span className="text-gray-500">Kaynak ID:</span>
                  <p className="font-mono text-xs break-all">
                    {selectedEntry.resource_id || "-"}
                  </p>
                </div>
              </div>
              <div>
                <span className="text-sm text-gray-500">Detaylar (JSON):</span>
                <pre className="mt-1 max-h-64 overflow-auto rounded-lg bg-gray-100 p-3 font-mono text-xs dark:bg-gray-800">
                  {JSON.stringify(selectedEntry.details, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
