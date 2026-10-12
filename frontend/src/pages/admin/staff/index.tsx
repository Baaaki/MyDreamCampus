import { useState, useMemo } from "react"
import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { staffApi } from "@/lib/api-client"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import {
  Plus,
  Pencil,
  Trash2,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Check,
  ChevronsUpDown,
} from "lucide-react"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { useFaculties } from "@/lib/services/catalog-service"
import { apiErrorMessage } from "@/lib/api-error"
import type { Staff } from "@/lib/types"
import { StaffDeleteDialog, StaffEditDialog } from "./staff-dialogs"

interface StaffListResponse {
  data: Staff[]
  pagination: {
    page: number
    limit: number
    total: number
    total_pages: number
  }
}

type SortField =
  | "first_name"
  | "last_name"
  | "email"
  | "role"
  | "department"
  | "office_location"
type SortDirection = "asc" | "desc"

// The staff API creates teacher accounts only (CreateStaffRequest is
// oneof=teacher): every new staff row becomes a teacher login in auth.
const STAFF_ROLE = "teacher"
const ROLE_LABELS: Record<string, string> = {
  teacher: "Öğretmen",
}

const STATUS_LABELS: Record<string, string> = {
  active: "Aktif",
  inactive: "Pasif",
}

export default function StaffPage() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [editingStaff, setEditingStaff] = useState<Staff | null>(null)
  const [deletingStaff, setDeletingStaff] = useState<Staff | null>(null)
  const [page, setPage] = useState(1)
  const [sortField, setSortField] = useState<SortField>("first_name")
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc")

  // UI state for faculty/department selection (faculty is only for filtering, not sent to backend)
  const [selectedFaculty, setSelectedFaculty] = useState("")
  const [facultyOpen, setFacultyOpen] = useState(false)
  const [departmentOpen, setDepartmentOpen] = useState(false)

  // Form states - Backend DTO'ya uygun alanlar
  const [formData, setFormData] = useState({
    email: "",
    first_name: "",
    last_name: "",
    department: "",
    phone: "",
    office_location: "",
  })

  const {
    data: faculties = [],
    isLoading: isLoadingFaculties,
    isError: isErrorFaculties,
  } = useFaculties()

  // Filtered departments based on selected faculty (for create form)
  const filteredDepartments = useMemo(() => {
    if (!selectedFaculty) return []
    const faculty = faculties.find((f) => f.name === selectedFaculty)
    return faculty?.departments || []
  }, [selectedFaculty, faculties])

  // Staff list — the previous page stays on screen while the next loads
  const {
    data: staffPage,
    isFetching: listLoading,
    error: listError,
    refetch: refetchStaff,
  } = useQuery({
    queryKey: ["staff", "list", page],
    queryFn: () =>
      staffApi.get(`?page=${page}&limit=10`).json<StaffListResponse>(),
    placeholderData: keepPreviousData,
  })
  const staffList: Staff[] = staffPage?.data ?? []
  const totalPages = staffPage?.pagination.total_pages ?? 1
  const displayError =
    error ||
    (listError
      ? apiErrorMessage(listError, "Personel listesi yüklenemedi")
      : "")

  // Sorting fonksiyonu
  const handleSort = (field: SortField) => {
    if (sortField === field) {
      // Aynı sütuna tıklandıysa direction değiştir
      setSortDirection(sortDirection === "asc" ? "desc" : "asc")
    } else {
      // Farklı sütuna tıklandıysa yeni field ve asc
      setSortField(field)
      setSortDirection("asc")
    }
  }

  // Sıralanmış staff listesi
  const sortedStaffList = [...staffList].sort((a, b) => {
    let aValue = a[sortField] || ""
    let bValue = b[sortField] || ""

    // String comparison
    if (typeof aValue === "string" && typeof bValue === "string") {
      aValue = aValue.toLowerCase()
      bValue = bValue.toLowerCase()
    }

    if (sortDirection === "asc") {
      return aValue > bValue ? 1 : -1
    } else {
      return aValue < bValue ? 1 : -1
    }
  })

  // Create staff
  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")

    try {
      // Backend DTO'ya uygun payload - sadece kabul edilen alanlar
      const payload = {
        email: formData.email,
        first_name: formData.first_name,
        last_name: formData.last_name,
        role: STAFF_ROLE,
        department: formData.department,
        phone: formData.phone,
        office_location: formData.office_location,
      }

      console.log("[Staff] Creating staff with payload:", payload)

      const response = await staffApi.post("", { json: payload }).json()
      console.log("[Staff] Create response:", response)

      setCreateDialogOpen(false)
      setFormData({
        email: "",
        first_name: "",
        last_name: "",
        department: "",
        phone: "",
        office_location: "",
      })
      setSelectedFaculty("")
      refetchStaff()
    } catch (err) {
      console.error("[Staff] Create error:", err)
      setError(apiErrorMessage(err, "Personel oluşturulamadı"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-background p-8">
      <div className="mx-auto max-w-7xl space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Personel Yönetimi</h1>
            <p className="mt-1 text-muted-foreground">
              Personel kayıtlarını ve bilgilerini yönetin
            </p>
          </div>

          {/* Create Dialog */}
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Yeni Personel Ekle
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[500px]">
              <DialogHeader>
                <DialogTitle>Yeni Personel Ekle</DialogTitle>
              </DialogHeader>
              <form onSubmit={handleCreate} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email">E-posta *</Label>
                  <Input
                    id="email"
                    type="email"
                    required
                    value={formData.email}
                    onChange={(e) =>
                      setFormData({ ...formData, email: e.target.value })
                    }
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="first_name">Ad *</Label>
                    <Input
                      id="first_name"
                      required
                      value={formData.first_name}
                      onChange={(e) =>
                        setFormData({ ...formData, first_name: e.target.value })
                      }
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="last_name">Soyad *</Label>
                    <Input
                      id="last_name"
                      required
                      value={formData.last_name}
                      onChange={(e) =>
                        setFormData({ ...formData, last_name: e.target.value })
                      }
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="role">Rol</Label>
                  <Input id="role" value={ROLE_LABELS[STAFF_ROLE]} disabled />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="faculty">Fakülte (filtreleme için)</Label>
                  <Popover open={facultyOpen} onOpenChange={setFacultyOpen}>
                    <PopoverTrigger asChild>
                      <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={facultyOpen}
                        className="w-full justify-between"
                      >
                        {selectedFaculty || "Fakülte seçin..."}
                        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent
                      className="w-[--radix-popover-trigger-width] p-0"
                      align="start"
                    >
                      <Command>
                        <CommandInput placeholder="Fakülte ara..." />
                        <CommandList>
                          {isLoadingFaculties && (
                            <div className="p-2 text-center text-sm text-muted-foreground">
                              Fakülteler yükleniyor...
                            </div>
                          )}
                          {isErrorFaculties && (
                            <div className="p-2 text-center text-sm text-destructive">
                              Fakülteler yüklenirken hata oluştu
                            </div>
                          )}
                          <CommandEmpty>Fakülte bulunamadı.</CommandEmpty>
                          <CommandGroup>
                            {faculties.map((faculty) => (
                              <CommandItem
                                key={faculty.id}
                                value={faculty.name}
                                onSelect={() => {
                                  setSelectedFaculty(
                                    selectedFaculty === faculty.name
                                      ? ""
                                      : faculty.name
                                  )
                                  setFormData({ ...formData, department: "" })
                                  setFacultyOpen(false)
                                }}
                              >
                                <Check
                                  className={`mr-2 h-4 w-4 ${selectedFaculty === faculty.name ? "opacity-100" : "opacity-0"}`}
                                />
                                {faculty.name}
                              </CommandItem>
                            ))}
                          </CommandGroup>
                        </CommandList>
                      </Command>
                    </PopoverContent>
                  </Popover>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="department">Bölüm</Label>
                  <Popover
                    open={departmentOpen}
                    onOpenChange={setDepartmentOpen}
                  >
                    <PopoverTrigger asChild>
                      <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={departmentOpen}
                        className="w-full justify-between"
                        disabled={!selectedFaculty}
                      >
                        {formData.department ||
                          (selectedFaculty
                            ? "Bölüm seçin..."
                            : "Önce fakülte seçin")}
                        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent
                      className="w-[--radix-popover-trigger-width] p-0"
                      align="start"
                    >
                      <Command>
                        <CommandInput placeholder="Bölüm ara..." />
                        <CommandList>
                          <CommandEmpty>Bölüm bulunamadı.</CommandEmpty>
                          <CommandGroup>
                            {filteredDepartments.map((dept) => (
                              <CommandItem
                                key={dept.id}
                                value={dept.name}
                                onSelect={() => {
                                  setFormData({
                                    ...formData,
                                    department:
                                      formData.department === dept.name
                                        ? ""
                                        : dept.name,
                                  })
                                  setDepartmentOpen(false)
                                }}
                              >
                                <Check
                                  className={`mr-2 h-4 w-4 ${formData.department === dept.name ? "opacity-100" : "opacity-0"}`}
                                />
                                {dept.name}
                              </CommandItem>
                            ))}
                          </CommandGroup>
                        </CommandList>
                      </Command>
                    </PopoverContent>
                  </Popover>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="phone">Telefon</Label>
                  <Input
                    id="phone"
                    value={formData.phone}
                    onChange={(e) =>
                      setFormData({ ...formData, phone: e.target.value })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="office_location">Ofis Konumu</Label>
                  <Input
                    id="office_location"
                    value={formData.office_location}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        office_location: e.target.value,
                      })
                    }
                  />
                </div>
                <div className="flex gap-2 pt-4">
                  <Button type="submit" disabled={loading} className="flex-1">
                    {loading ? "Oluşturuluyor..." : "Personel Oluştur"}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setCreateDialogOpen(false)}
                    className="flex-1"
                  >
                    İptal
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        {/* Error Alert */}
        {displayError && (
          <div className="rounded-lg border border-destructive bg-destructive/10 px-4 py-3 text-destructive">
            {displayError}
          </div>
        )}

        {/* Staff Table */}
        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort("first_name")}
                    className="h-8 px-2"
                  >
                    Ad Soyad
                    {sortField === "first_name" &&
                      (sortDirection === "asc" ? (
                        <ArrowUp className="ml-2 h-4 w-4" />
                      ) : (
                        <ArrowDown className="ml-2 h-4 w-4" />
                      ))}
                    {sortField !== "first_name" && (
                      <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />
                    )}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort("email")}
                    className="h-8 px-2"
                  >
                    E-posta
                    {sortField === "email" &&
                      (sortDirection === "asc" ? (
                        <ArrowUp className="ml-2 h-4 w-4" />
                      ) : (
                        <ArrowDown className="ml-2 h-4 w-4" />
                      ))}
                    {sortField !== "email" && (
                      <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />
                    )}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort("role")}
                    className="h-8 px-2"
                  >
                    Rol
                    {sortField === "role" &&
                      (sortDirection === "asc" ? (
                        <ArrowUp className="ml-2 h-4 w-4" />
                      ) : (
                        <ArrowDown className="ml-2 h-4 w-4" />
                      ))}
                    {sortField !== "role" && (
                      <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />
                    )}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort("department")}
                    className="h-8 px-2"
                  >
                    Bölüm
                    {sortField === "department" &&
                      (sortDirection === "asc" ? (
                        <ArrowUp className="ml-2 h-4 w-4" />
                      ) : (
                        <ArrowDown className="ml-2 h-4 w-4" />
                      ))}
                    {sortField !== "department" && (
                      <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />
                    )}
                  </Button>
                </TableHead>
                <TableHead>Telefon</TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleSort("office_location")}
                    className="h-8 px-2"
                  >
                    Ofis
                    {sortField === "office_location" &&
                      (sortDirection === "asc" ? (
                        <ArrowUp className="ml-2 h-4 w-4" />
                      ) : (
                        <ArrowDown className="ml-2 h-4 w-4" />
                      ))}
                    {sortField !== "office_location" && (
                      <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />
                    )}
                  </Button>
                </TableHead>
                <TableHead>Durum</TableHead>
                <TableHead className="text-right">İşlemler</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(loading || listLoading) && staffList.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={8}
                    className="text-center text-muted-foreground"
                  >
                    Yükleniyor...
                  </TableCell>
                </TableRow>
              ) : staffList.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={8}
                    className="text-center text-muted-foreground"
                  >
                    Personel bulunamadı
                  </TableCell>
                </TableRow>
              ) : (
                sortedStaffList.map((staff) => (
                  <TableRow key={staff.id}>
                    <TableCell className="font-medium">
                      {staff.first_name} {staff.last_name}
                    </TableCell>
                    <TableCell>{staff.email}</TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {ROLE_LABELS[staff.role] ?? staff.role}
                      </Badge>
                    </TableCell>
                    <TableCell>{staff.department || "-"}</TableCell>
                    <TableCell>{staff.phone || "-"}</TableCell>
                    <TableCell>{staff.office_location || "-"}</TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          staff.status === "active" ? "default" : "destructive"
                        }
                      >
                        {STATUS_LABELS[staff.status] ?? staff.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="space-x-2 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={`${staff.first_name} ${staff.last_name} düzenle`}
                        onClick={() => setEditingStaff(staff)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={`${staff.first_name} ${staff.last_name} sil`}
                        onClick={() => setDeletingStaff(staff)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
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
          <div className="flex justify-center gap-2">
            <Button
              variant="outline"
              onClick={() => setPage(page - 1)}
              disabled={page === 1}
            >
              Önceki
            </Button>
            <span className="flex items-center px-4">
              Sayfa {page} / {totalPages}
            </span>
            <Button
              variant="outline"
              onClick={() => setPage(page + 1)}
              disabled={page === totalPages}
            >
              Sonraki
            </Button>
          </div>
        )}

        <StaffEditDialog
          staff={editingStaff}
          onOpenChange={(open) => !open && setEditingStaff(null)}
        />
        <StaffDeleteDialog
          staff={deletingStaff}
          onOpenChange={(open) => !open && setDeletingStaff(null)}
        />
      </div>
    </div>
  )
}
