import { useState } from "react"
import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { studentApi } from "@/lib/api-client"
import { useFaculties } from "@/lib/services/catalog-service"
import { toCreateStudentPayload } from "@/lib/services/admin-student-service"
import { staffApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import type { Department, Staff } from "@/lib/types"
import {
  ArrowUp,
  ArrowDown,
  ArrowUpDown,
  Plus,
  Users,
  Loader2,
  AlertCircle,
} from "lucide-react"
import { Link } from "react-router"
import { StudentImportDialog } from "./import-dialog"

type Student = {
  id: string
  student_number: string
  first_name: string
  last_name: string
  email: string
  faculty: string
  department: string
  enrollment_year: number
  class_level: number
  advisor_id?: string
  advisor_name?: string
  advisor?: { first_name: string; last_name: string }
  status: string
  created_at: string
  updated_at: string
}

type StudentListResponse = {
  data: Student[]
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
  | "student_number"
  | "department"
  | "faculty"
  | "enrollment_year"
  | "class_level"
  | "status"
type SortDirection = "asc" | "desc"

const STATUS_LABELS: Record<string, string> = {
  active: "Aktif",
  graduated: "Mezun",
  suspended: "Askıda",
  withdrawn: "Kaydı Silinmiş",
}

export default function StudentsPage() {
  const [currentPage, setCurrentPage] = useState(1)
  const [limit] = useState(10)
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isEditOpen, setIsEditOpen] = useState(false)
  const [editingStudent, setEditingStudent] = useState<Student | null>(null)
  const [sortField, setSortField] = useState<SortField>("first_name")
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc")

  // Faculty/Department selection
  const [createDepartments, setCreateDepartments] = useState<Department[]>([])

  // Create form state
  const [createFormData, setCreateFormData] = useState({
    student_number: "",
    first_name: "",
    last_name: "",
    email: "",
    faculty: "",
    department: "",
    enrollment_year: new Date().getFullYear(),
    class_level: 1,
    advisor_id: "", // Will be selected from dropdown
  })

  // Update form state
  const [updateFormData, setUpdateFormData] = useState({
    student_number: "",
    first_name: "",
    last_name: "",
    email: "",
    faculty: "",
    department: "",
    enrollment_year: new Date().getFullYear(),
    class_level: 1,
    status: "active",
    advisor_id: "",
  })

  // Department-specific advisors for edit form
  const [editDepartmentAdvisors, setEditDepartmentAdvisors] = useState<Staff[]>(
    []
  )
  const [loadingEditAdvisors, setLoadingEditAdvisors] = useState(false)

  const {
    data: faculties = [],
    isLoading: isLoadingFaculties,
    isError: isErrorFaculties,
  } = useFaculties()

  const {
    data: studentPage,
    isFetching: loading,
    refetch: fetchStudents,
  } = useQuery({
    queryKey: ["students", "list", currentPage, limit],
    queryFn: () =>
      studentApi
        .get("", {
          searchParams: {
            page: currentPage.toString(),
            limit: limit.toString(),
          },
        })
        .json<StudentListResponse>(),
    placeholderData: keepPreviousData,
  })
  const studentList = studentPage?.data ?? []
  const totalPages = studentPage?.pagination.total_pages ?? 1

  // Fetch department-specific advisors when department changes in create form
  const fetchAdvisorsByDepartment = async (
    department: string
  ): Promise<Staff[]> => {
    try {
      const response = (await staffApi
        .get("instructors", { searchParams: { department } })
        .json()) as { data: Staff[] }
      return response.data || []
    } catch (err) {
      console.error("Error fetching advisors by department:", err)
      // Hata durumunda boş dön - fallback yok
      return []
    }
  }

  // Department-specific advisors for create form; empty when the department has none
  const { data: departmentAdvisors = [], isFetching: loadingAdvisors } =
    useQuery({
      queryKey: ["staff", "instructors", createFormData.department],
      queryFn: () => fetchAdvisorsByDepartment(createFormData.department),
      enabled: !!createFormData.department,
    })

  const handleCreateStudent = async (e: React.FormEvent) => {
    e.preventDefault()
    console.log("[Students Page] handleCreateStudent called")
    console.log("[Students Page] createFormData:", createFormData)
    try {
      console.log("[Students Page] Calling studentApi.post...")
      const response = await studentApi
        .post("", { json: toCreateStudentPayload(createFormData) })
        .json()
      console.log("[Students Page] POST Response:", response)
      setIsCreateOpen(false)
      setCreateFormData({
        student_number: "",
        first_name: "",
        last_name: "",
        email: "",
        faculty: "",
        department: "",
        enrollment_year: new Date().getFullYear(),
        class_level: 1,
        advisor_id: "",
      })
      setCreateDepartments([])
      fetchStudents()
    } catch (error) {
      console.error("[Students Page] Failed to create student:", error)
      alert(apiErrorMessage(error, "Öğrenci oluşturulamadı"))
    }
  }

  const handleUpdateStudent = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingStudent) return

    try {
      // Backend only accepts class_level, advisor_id, status for updates
      const payload: {
        class_level: number
        status: string
        advisor_id?: string
      } = {
        class_level: updateFormData.class_level,
        status: updateFormData.status,
      }

      // Only include advisor_id if it's set
      if (updateFormData.advisor_id) {
        payload.advisor_id = updateFormData.advisor_id
      }

      await studentApi.put(`${editingStudent.id}`, { json: payload })
      setIsEditOpen(false)
      setEditingStudent(null)
      setEditDepartmentAdvisors([])
      fetchStudents()
    } catch (error) {
      console.error("Failed to update student:", error)
      alert(apiErrorMessage(error, "Öğrenci güncellenemedi"))
    }
  }

  const handleDeleteStudent = async (id: string) => {
    if (!confirm("Bu öğrenciyi silmek istediğinize emin misiniz?")) return

    try {
      await studentApi.delete(`${id}`)
      fetchStudents()
    } catch (error) {
      console.error("Failed to delete student:", error)
      alert(apiErrorMessage(error, "Öğrenci silinemedi"))
    }
  }

  const openEditModal = (student: Student) => {
    setEditingStudent(student)
    setUpdateFormData({
      student_number: student.student_number,
      first_name: student.first_name,
      last_name: student.last_name,
      email: student.email,
      faculty: student.faculty,
      department: student.department,
      enrollment_year: student.enrollment_year,
      class_level: student.class_level,
      status: student.status,
      advisor_id: student.advisor_id || "",
    })
    // Fetch advisors for student's department
    if (student.department) {
      setLoadingEditAdvisors(true)
      fetchAdvisorsByDepartment(student.department)
        .then((instructors) => {
          setEditDepartmentAdvisors(instructors)
        })
        .finally(() => {
          setLoadingEditAdvisors(false)
        })
    }
    setIsEditOpen(true)
  }

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc")
    } else {
      setSortField(field)
      setSortDirection("asc")
    }
  }

  const getSortIcon = (field: SortField) => {
    if (sortField !== field) {
      return <ArrowUpDown className="ml-2 h-4 w-4" />
    }
    return sortDirection === "asc" ? (
      <ArrowUp className="ml-2 h-4 w-4" />
    ) : (
      <ArrowDown className="ml-2 h-4 w-4" />
    )
  }

  const sortedStudentList = [...studentList].sort((a, b) => {
    let aValue: string | number = a[sortField] || ""
    let bValue: string | number = b[sortField] || ""

    if (typeof aValue === "string") {
      aValue = aValue.toLowerCase()
      bValue = (bValue as string).toLowerCase()
    }

    if (sortDirection === "asc") {
      return aValue > bValue ? 1 : -1
    } else {
      return aValue < bValue ? 1 : -1
    }
  })

  return (
    <div className="container mx-auto py-10">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-3xl font-bold">Öğrenci Yönetimi</h1>
        <div className="flex gap-2">
          <Link to="/students/advisors">
            <Button variant="outline">
              <Users className="mr-2 h-4 w-4" /> Danışman İşleri
            </Button>
          </Link>
          <StudentImportDialog />

          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" /> Yeni Öğrenci Ekle
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Yeni Öğrenci Oluştur</DialogTitle>
                <DialogDescription>
                  Sisteme yeni bir öğrenci ekleyin.
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={handleCreateStudent} className="space-y-4">
                <div>
                  <Label htmlFor="student_number">Öğrenci No</Label>
                  <Input
                    id="student_number"
                    value={createFormData.student_number}
                    onChange={(e) =>
                      setCreateFormData({
                        ...createFormData,
                        student_number: e.target.value,
                      })
                    }
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="first_name">Ad</Label>
                  <Input
                    id="first_name"
                    value={createFormData.first_name}
                    onChange={(e) =>
                      setCreateFormData({
                        ...createFormData,
                        first_name: e.target.value,
                      })
                    }
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="last_name">Soyad</Label>
                  <Input
                    id="last_name"
                    value={createFormData.last_name}
                    onChange={(e) =>
                      setCreateFormData({
                        ...createFormData,
                        last_name: e.target.value,
                      })
                    }
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="email">E-posta</Label>
                  <Input
                    id="email"
                    type="email"
                    value={createFormData.email}
                    onChange={(e) =>
                      setCreateFormData({
                        ...createFormData,
                        email: e.target.value,
                      })
                    }
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="faculty">Fakülte</Label>
                  <Select
                    value={createFormData.faculty}
                    onValueChange={(value) => {
                      const selectedFaculty = faculties.find(
                        (f) => f.name === value
                      )
                      setCreateFormData({
                        ...createFormData,
                        faculty: value,
                        department: "", // Reset department when faculty changes
                      })
                      setCreateDepartments(selectedFaculty?.departments || [])
                    }}
                    disabled={isLoadingFaculties}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue
                        placeholder={
                          isLoadingFaculties
                            ? "Fakülteler yükleniyor..."
                            : isErrorFaculties
                              ? "Fakülteler yüklenemedi"
                              : "Fakülte seçin..."
                        }
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {faculties.map((faculty) => (
                        <SelectItem key={faculty.id} value={faculty.name}>
                          {faculty.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {isErrorFaculties && (
                    <p className="mt-1 flex items-center text-xs text-destructive">
                      <AlertCircle className="mr-1 h-3 w-3" />
                      Fakülteler yüklenirken bir hata oluştu
                    </p>
                  )}
                </div>
                <div>
                  <Label htmlFor="department">Bölüm</Label>
                  <Select
                    value={createFormData.department}
                    onValueChange={(value) =>
                      setCreateFormData({
                        ...createFormData,
                        department: value,
                        advisor_id: "", // Advisors are per department
                      })
                    }
                    disabled={!createFormData.faculty}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue
                        placeholder={
                          createFormData.faculty
                            ? "Bölüm seçin..."
                            : "Önce fakülte seçin"
                        }
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {createDepartments.map((dept) => (
                        <SelectItem key={dept.id} value={dept.name}>
                          {dept.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="advisor">
                    Danışman
                    {loadingAdvisors && (
                      <Loader2 className="ml-2 inline h-4 w-4 animate-spin" />
                    )}
                  </Label>
                  {loadingAdvisors ? (
                    <div className="mt-1.5 rounded-md border p-2 text-sm text-muted-foreground">
                      Danışmanlar yükleniyor...
                    </div>
                  ) : (
                    <Select
                      value={createFormData.advisor_id}
                      onValueChange={(value) =>
                        setCreateFormData({
                          ...createFormData,
                          advisor_id: value,
                        })
                      }
                      disabled={!createFormData.department}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue
                          placeholder={
                            createFormData.department
                              ? "Danışman seçin..."
                              : "Önce bölüm seçin"
                          }
                        />
                      </SelectTrigger>
                      <SelectContent>
                        {departmentAdvisors.length === 0 ? (
                          <SelectItem value="no-advisors" disabled>
                            Bu bölümde danışman bulunamadı
                          </SelectItem>
                        ) : (
                          departmentAdvisors.map((staff) => (
                            <SelectItem key={staff.id} value={staff.id}>
                              {staff.first_name} {staff.last_name}
                            </SelectItem>
                          ))
                        )}
                      </SelectContent>
                    </Select>
                  )}
                  {createFormData.department &&
                    !loadingAdvisors &&
                    departmentAdvisors.length > 0 && (
                      <p className="mt-1 text-xs text-green-600">
                        {departmentAdvisors.length} danışman bulundu
                      </p>
                    )}
                </div>
                <div>
                  <Label htmlFor="enrollment_year">Kayıt Yılı</Label>
                  <Input
                    id="enrollment_year"
                    type="number"
                    value={createFormData.enrollment_year}
                    onChange={(e) =>
                      setCreateFormData({
                        ...createFormData,
                        enrollment_year: parseInt(e.target.value),
                      })
                    }
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="class_level">Sınıf</Label>
                  <Input
                    id="class_level"
                    type="number"
                    min="1"
                    max="6"
                    value={createFormData.class_level}
                    onChange={(e) =>
                      setCreateFormData({
                        ...createFormData,
                        class_level: parseInt(e.target.value),
                      })
                    }
                    required
                  />
                </div>
                <div className="flex justify-end space-x-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setIsCreateOpen(false)}
                  >
                    İptal
                  </Button>
                  <Button type="submit">Oluştur</Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {loading ? (
        <p>Yükleniyor...</p>
      ) : (
        <>
          <Table>
            <TableCaption>Sistemdeki tüm öğrenciler.</TableCaption>
            <TableHeader>
              <TableRow>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort("student_number")}
                    className="flex items-center"
                  >
                    Öğrenci No
                    {getSortIcon("student_number")}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort("first_name")}
                    className="flex items-center"
                  >
                    Ad Soyad
                    {getSortIcon("first_name")}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort("email")}
                    className="flex items-center"
                  >
                    E-posta
                    {getSortIcon("email")}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort("faculty")}
                    className="flex items-center"
                  >
                    Fakülte
                    {getSortIcon("faculty")}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort("department")}
                    className="flex items-center"
                  >
                    Bölüm
                    {getSortIcon("department")}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort("enrollment_year")}
                    className="flex items-center"
                  >
                    Kayıt Yılı
                    {getSortIcon("enrollment_year")}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort("class_level")}
                    className="flex items-center"
                  >
                    Sınıf
                    {getSortIcon("class_level")}
                  </Button>
                </TableHead>
                <TableHead>Danışman</TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort("status")}
                    className="flex items-center"
                  >
                    Durum
                    {getSortIcon("status")}
                  </Button>
                </TableHead>
                <TableHead>İşlemler</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedStudentList.map((student) => (
                <TableRow key={student.id}>
                  <TableCell className="font-medium">
                    {student.student_number}
                  </TableCell>
                  <TableCell>
                    {student.first_name} {student.last_name}
                  </TableCell>
                  <TableCell>{student.email}</TableCell>
                  <TableCell>{student.faculty}</TableCell>
                  <TableCell>{student.department}</TableCell>
                  <TableCell>{student.enrollment_year}</TableCell>
                  <TableCell>{student.class_level}</TableCell>
                  <TableCell>
                    {student.advisor_name || student.advisor ? (
                      <span className="text-sm">
                        {student.advisor_name ||
                          `${student.advisor?.first_name} ${student.advisor?.last_name}`}
                      </span>
                    ) : (
                      <span className="text-sm text-muted-foreground">
                        Danışman Yok
                      </span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        student.status === "active"
                          ? "default"
                          : student.status === "graduated"
                            ? "secondary"
                            : "destructive"
                      }
                    >
                      {STATUS_LABELS[student.status] ?? student.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex space-x-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => openEditModal(student)}
                      >
                        Düzenle
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDeleteStudent(student.id)}
                      >
                        Sil
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <div className="mt-4 flex items-center justify-between">
            <Button
              variant="outline"
              onClick={() => setCurrentPage((prev) => Math.max(prev - 1, 1))}
              disabled={currentPage === 1}
            >
              Önceki
            </Button>
            <span>
              Sayfa {currentPage} / {totalPages}
            </span>
            <Button
              variant="outline"
              onClick={() =>
                setCurrentPage((prev) => Math.min(prev + 1, totalPages))
              }
              disabled={currentPage === totalPages}
            >
              Sonraki
            </Button>
          </div>
        </>
      )}

      {/* Edit Dialog */}
      <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Öğrenciyi Düzenle</DialogTitle>
            <DialogDescription>
              Öğrenci bilgilerini güncelleyin. Not: Yalnızca sınıf, durum ve
              danışman değiştirilebilir.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleUpdateStudent} className="space-y-4">
            <div>
              <Label htmlFor="edit_student_number">Öğrenci No</Label>
              <Input
                id="edit_student_number"
                value={updateFormData.student_number}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_first_name">Ad</Label>
              <Input
                id="edit_first_name"
                value={updateFormData.first_name}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_last_name">Soyad</Label>
              <Input
                id="edit_last_name"
                value={updateFormData.last_name}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_email">E-posta</Label>
              <Input
                id="edit_email"
                type="email"
                value={updateFormData.email}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_faculty">Fakülte</Label>
              <Input
                id="edit_faculty"
                value={updateFormData.faculty}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_department">Bölüm</Label>
              <Input
                id="edit_department"
                value={updateFormData.department}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_enrollment_year">Kayıt Yılı</Label>
              <Input
                id="edit_enrollment_year"
                type="number"
                value={updateFormData.enrollment_year}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_class_level">Sınıf</Label>
              <Input
                id="edit_class_level"
                type="number"
                min="1"
                max="6"
                value={updateFormData.class_level}
                onChange={(e) =>
                  setUpdateFormData({
                    ...updateFormData,
                    class_level: parseInt(e.target.value),
                  })
                }
                required
              />
            </div>
            <div>
              <Label htmlFor="edit_status">Durum</Label>
              <select
                id="edit_status"
                value={updateFormData.status}
                onChange={(e) =>
                  setUpdateFormData({
                    ...updateFormData,
                    status: e.target.value,
                  })
                }
                className="w-full rounded-md border p-2"
                required
              >
                <option value="active">Aktif</option>
                <option value="graduated">Mezun</option>
                <option value="suspended">Askıda</option>
                <option value="withdrawn">Kaydı Silinmiş</option>
              </select>
            </div>
            <div>
              <Label htmlFor="edit_advisor">
                Danışman
                {loadingEditAdvisors && (
                  <Loader2 className="ml-2 inline h-4 w-4 animate-spin" />
                )}
              </Label>
              {loadingEditAdvisors ? (
                <div className="mt-1.5 rounded-md border p-2 text-sm text-muted-foreground">
                  Danışmanlar yükleniyor...
                </div>
              ) : (
                <Select
                  value={updateFormData.advisor_id}
                  onValueChange={(value) =>
                    setUpdateFormData({ ...updateFormData, advisor_id: value })
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Danışman seçin..." />
                  </SelectTrigger>
                  <SelectContent>
                    {editDepartmentAdvisors.length === 0 ? (
                      <SelectItem value="no-advisors" disabled>
                        Bu bölümde danışman bulunamadı
                      </SelectItem>
                    ) : (
                      editDepartmentAdvisors.map((staff) => (
                        <SelectItem key={staff.id} value={staff.id}>
                          {staff.first_name} {staff.last_name}
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
              )}
              {!loadingEditAdvisors && editDepartmentAdvisors.length > 0 && (
                <p className="mt-1 text-xs text-green-600">
                  {editDepartmentAdvisors.length} danışman bulundu (
                  {updateFormData.department})
                </p>
              )}
            </div>
            <div className="flex justify-end space-x-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsEditOpen(false)}
              >
                İptal
              </Button>
              <Button type="submit">Güncelle</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
