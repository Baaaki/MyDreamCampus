'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { studentApi } from '@/lib/api-client'
import { mockFaculties } from '@/mock_data/catalog'
import { staffApi } from '@/lib/api-client'
import type { Department, Staff } from '@/lib/types'
import { ArrowUp, ArrowDown, ArrowUpDown, Plus, Upload, Users, Loader2 } from 'lucide-react'
import Link from 'next/link'

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

type SortField = 'first_name' | 'last_name' | 'email' | 'student_number' | 'department' | 'faculty' | 'enrollment_year' | 'class_level' | 'status'
type SortDirection = 'asc' | 'desc'

export default function StudentsPage() {
  const [studentList, setStudentList] = useState<Student[]>([])
  const [loading, setLoading] = useState(false)
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [limit] = useState(10)
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isEditOpen, setIsEditOpen] = useState(false)
  const [isImportOpen, setIsImportOpen] = useState(false)
  const [editingStudent, setEditingStudent] = useState<Student | null>(null)
  const [sortField, setSortField] = useState<SortField>('first_name')
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  
  // Faculty/Department selection
  const [createDepartments, setCreateDepartments] = useState<Department[]>([])
  const [editDepartments, setEditDepartments] = useState<Department[]>([])
  
  // Advisors from staff API
  const [advisors, setAdvisors] = useState<Staff[]>([])

  // Department-specific advisors for create form
  const [departmentAdvisors, setDepartmentAdvisors] = useState<Staff[]>([])
  const [loadingAdvisors, setLoadingAdvisors] = useState(false)

  // Create form state
  const [createFormData, setCreateFormData] = useState({
    student_number: '',
    first_name: '',
    last_name: '',
    email: '',
    faculty: '',
    department: '',
    enrollment_year: new Date().getFullYear(),
    class_level: 1,
    advisor_id: '', // Will be selected from dropdown
  })

  // Update form state
  const [updateFormData, setUpdateFormData] = useState({
    student_number: '',
    first_name: '',
    last_name: '',
    email: '',
    faculty: '',
    department: '',
    enrollment_year: new Date().getFullYear(),
    class_level: 1,
    status: 'active',
    advisor_id: '',
  })

  // Department-specific advisors for edit form
  const [editDepartmentAdvisors, setEditDepartmentAdvisors] = useState<Staff[]>([])
  const [loadingEditAdvisors, setLoadingEditAdvisors] = useState(false)

  useEffect(() => {
    fetchStudents()
  }, [currentPage, limit])

  // Fetch all advisors (teachers) from staff API
  useEffect(() => {
    const fetchAdvisors = async () => {
      try {
        const response = await staffApi.get('', { searchParams: { role: 'teacher', limit: '100' } }).json() as { data: Staff[] }
        setAdvisors(response.data || [])
      } catch (error) {
        console.error('Failed to fetch advisors:', error)
      }
    }
    fetchAdvisors()
  }, [])

  // Fetch department-specific advisors when department changes in create form
  const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost'

  const fetchAdvisorsByDepartment = async (department: string): Promise<Staff[]> => {
    try {
      const response = await fetch(
        `${API_BASE_URL}:8002/internal/staff/instructors?department=${encodeURIComponent(department)}`
      )

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const data = await response.json()
      const instructors = data.data || []
      console.log(`[Students] Fetched ${instructors.length} advisors for department: ${department}`)
      return instructors
    } catch (err) {
      console.error('Error fetching advisors by department:', err)
      // Hata durumunda boş dön - fallback yok
      return []
    }
  }

  // Effect to load advisors when department is selected in create form
  useEffect(() => {
    if (createFormData.department) {
      setLoadingAdvisors(true)
      setCreateFormData(prev => ({ ...prev, advisor_id: '' })) // Reset advisor when department changes
      fetchAdvisorsByDepartment(createFormData.department)
        .then((instructors) => {
          setDepartmentAdvisors(instructors) // Bölümde hoca yoksa boş kalacak
        })
        .finally(() => {
          setLoadingAdvisors(false)
        })
    } else {
      setDepartmentAdvisors([])
    }
  }, [createFormData.department])

  const fetchStudents = async () => {
    setLoading(true)
    console.log('[Students Page] Fetching students, page:', currentPage, 'limit:', limit)
    try {
      const response = (await studentApi
        .get('', {
          searchParams: {
            page: currentPage.toString(),
            limit: limit.toString(),
          },
        })
        .json()) as StudentListResponse

      console.log('[Students Page] Response:', response)
      setStudentList(response.data)
      setTotalPages(response.pagination.total_pages)
    } catch (error) {
      console.error('Failed to fetch students:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateStudent = async (e: React.FormEvent) => {
    e.preventDefault()
    console.log('[Students Page] handleCreateStudent called')
    console.log('[Students Page] createFormData:', createFormData)
    try {
      console.log('[Students Page] Calling studentApi.post...')
      const response = await studentApi.post('', { json: createFormData }).json()
      console.log('[Students Page] POST Response:', response)
      setIsCreateOpen(false)
      setCreateFormData({
        student_number: '',
        first_name: '',
        last_name: '',
        email: '',
        faculty: '',
        department: '',
        enrollment_year: new Date().getFullYear(),
        class_level: 1,
        advisor_id: '',
      })
      setCreateDepartments([])
      setDepartmentAdvisors([])
      fetchStudents()
    } catch (error) {
      console.error('[Students Page] Failed to create student:', error)
    }
  }

  const handleUpdateStudent = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingStudent) return

    try {
      // Backend only accepts class_level, advisor_id, status for updates
      const payload: { class_level: number; status: string; advisor_id?: string } = {
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
      console.error('Failed to update student:', error)
    }
  }

  const handleDeleteStudent = async (id: string) => {
    if (!confirm('Are you sure you want to delete this student?')) return

    try {
      await studentApi.delete(`${id}`)
      fetchStudents()
    } catch (error) {
      console.error('Failed to delete student:', error)
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
      advisor_id: student.advisor_id || '',
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

  const handleBulkImport = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedFile) return

    setImporting(true)
    try {
      const formData = new FormData()
      formData.append('file', selectedFile)

      // Note: Mock API doesn't support FormData, this will work with real backend
      const response = await fetch('/api/v1/students/import', {
        method: 'POST',
        body: formData,
      })

      if (response.ok) {
        const result = await response.json()
        alert(`Import job created successfully. Job ID: ${result.job_id}`)
        setIsImportOpen(false)
        setSelectedFile(null)
        fetchStudents()
      } else {
        alert('Import failed')
      }
    } catch (error) {
      console.error('Failed to import students:', error)
      alert('Import failed: ' + error)
    } finally {
      setImporting(false)
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setSelectedFile(e.target.files[0])
    }
  }

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('asc')
    }
  }

  const getSortIcon = (field: SortField) => {
    if (sortField !== field) {
      return <ArrowUpDown className="ml-2 h-4 w-4" />
    }
    return sortDirection === 'asc' ? (
      <ArrowUp className="ml-2 h-4 w-4" />
    ) : (
      <ArrowDown className="ml-2 h-4 w-4" />
    )
  }

  const sortedStudentList = [...studentList].sort((a, b) => {
    let aValue: string | number = a[sortField] || ''
    let bValue: string | number = b[sortField] || ''

    if (typeof aValue === 'string') {
      aValue = aValue.toLowerCase()
      bValue = (bValue as string).toLowerCase()
    }

    if (sortDirection === 'asc') {
      return aValue > bValue ? 1 : -1
    } else {
      return aValue < bValue ? 1 : -1
    }
  })

  return (
    <div className="container mx-auto py-10">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Student Management</h1>
        <div className="flex gap-2">
          <Link href="/students/advisors">
            <Button variant="outline">
              <Users className="mr-2 h-4 w-4" /> Danışman İşleri
            </Button>
          </Link>
          <Dialog open={isImportOpen} onOpenChange={setIsImportOpen}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <Upload className="mr-2 h-4 w-4" /> Bulk Import
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Bulk Import Students</DialogTitle>
                <DialogDescription>
                  Upload a CSV file to import multiple students at once.
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={handleBulkImport} className="space-y-4">
                <div>
                  <Label htmlFor="csv_file">CSV File</Label>
                  <Input
                    id="csv_file"
                    type="file"
                    accept=".csv"
                    onChange={handleFileChange}
                    required
                  />
                  <p className="text-sm text-muted-foreground mt-2">
                    CSV should include: student_number, first_name, last_name, email, faculty, department, enrollment_year, class_level
                  </p>
                </div>
                <div className="flex justify-end space-x-2">
                  <Button type="button" variant="outline" onClick={() => setIsImportOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={importing || !selectedFile}>
                    {importing ? 'Importing...' : 'Import'}
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>

          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" /> Add New Student
              </Button>
            </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create New Student</DialogTitle>
              <DialogDescription>
                Add a new student to the system.
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleCreateStudent} className="space-y-4">
              <div>
                <Label htmlFor="student_number">Student Number</Label>
                <Input
                  id="student_number"
                  value={createFormData.student_number}
                  onChange={(e) =>
                    setCreateFormData({ ...createFormData, student_number: e.target.value })
                  }
                  required
                />
              </div>
              <div>
                <Label htmlFor="first_name">First Name</Label>
                <Input
                  id="first_name"
                  value={createFormData.first_name}
                  onChange={(e) =>
                    setCreateFormData({ ...createFormData, first_name: e.target.value })
                  }
                  required
                />
              </div>
              <div>
                <Label htmlFor="last_name">Last Name</Label>
                <Input
                  id="last_name"
                  value={createFormData.last_name}
                  onChange={(e) =>
                    setCreateFormData({ ...createFormData, last_name: e.target.value })
                  }
                  required
                />
              </div>
              <div>
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={createFormData.email}
                  onChange={(e) =>
                    setCreateFormData({ ...createFormData, email: e.target.value })
                  }
                  required
                />
              </div>
              <div>
                <Label htmlFor="faculty">Fakülte</Label>
                <Select
                  value={createFormData.faculty}
                  onValueChange={(value) => {
                    const selectedFaculty = mockFaculties.find(f => f.name === value)
                    setCreateFormData({ 
                      ...createFormData, 
                      faculty: value,
                      department: '' // Reset department when faculty changes
                    })
                    setCreateDepartments(selectedFaculty?.departments || [])
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Fakülte seçin..." />
                  </SelectTrigger>
                  <SelectContent>
                    {mockFaculties.map((faculty) => (
                      <SelectItem key={faculty.id} value={faculty.name}>
                        {faculty.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="department">Bölüm</Label>
                <Select
                  value={createFormData.department}
                  onValueChange={(value) =>
                    setCreateFormData({ ...createFormData, department: value })
                  }
                  disabled={!createFormData.faculty}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder={createFormData.faculty ? "Bölüm seçin..." : "Önce fakülte seçin"} />
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
                    <Loader2 className="ml-2 h-4 w-4 animate-spin inline" />
                  )}
                </Label>
                {loadingAdvisors ? (
                  <div className="mt-1.5 p-2 border rounded-md text-sm text-muted-foreground">
                    Danışmanlar yükleniyor...
                  </div>
                ) : (
                  <Select
                    value={createFormData.advisor_id}
                    onValueChange={(value) =>
                      setCreateFormData({ ...createFormData, advisor_id: value })
                    }
                    disabled={!createFormData.department}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder={createFormData.department ? "Danışman seçin..." : "Önce bölüm seçin"} />
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
                {createFormData.department && !loadingAdvisors && departmentAdvisors.length > 0 && (
                  <p className="mt-1 text-xs text-green-600">
                    {departmentAdvisors.length} danışman bulundu
                  </p>
                )}
              </div>
              <div>
                <Label htmlFor="enrollment_year">Enrollment Year</Label>
                <Input
                  id="enrollment_year"
                  type="number"
                  value={createFormData.enrollment_year}
                  onChange={(e) =>
                    setCreateFormData({ ...createFormData, enrollment_year: parseInt(e.target.value) })
                  }
                  required
                />
              </div>
              <div>
                <Label htmlFor="class_level">Class Level</Label>
                <Input
                  id="class_level"
                  type="number"
                  min="1"
                  max="6"
                  value={createFormData.class_level}
                  onChange={(e) =>
                    setCreateFormData({ ...createFormData, class_level: parseInt(e.target.value) })
                  }
                  required
                />
              </div>
              <div className="flex justify-end space-x-2">
                <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit">Create</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
        </div>
      </div>

      {loading ? (
        <p>Loading...</p>
      ) : (
        <>
          <Table>
            <TableCaption>A list of all students in the system.</TableCaption>
            <TableHeader>
              <TableRow>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort('student_number')}
                    className="flex items-center"
                  >
                    Student Number
                    {getSortIcon('student_number')}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort('first_name')}
                    className="flex items-center"
                  >
                    Name
                    {getSortIcon('first_name')}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort('email')}
                    className="flex items-center"
                  >
                    Email
                    {getSortIcon('email')}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort('faculty')}
                    className="flex items-center"
                  >
                    Faculty
                    {getSortIcon('faculty')}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort('department')}
                    className="flex items-center"
                  >
                    Department
                    {getSortIcon('department')}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort('enrollment_year')}
                    className="flex items-center"
                  >
                    Enrollment Year
                    {getSortIcon('enrollment_year')}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort('class_level')}
                    className="flex items-center"
                  >
                    Class Level
                    {getSortIcon('class_level')}
                  </Button>
                </TableHead>
                <TableHead>Advisor</TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    onClick={() => handleSort('status')}
                    className="flex items-center"
                  >
                    Status
                    {getSortIcon('status')}
                  </Button>
                </TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedStudentList.map((student) => (
                <TableRow key={student.id}>
                  <TableCell className="font-medium">{student.student_number}</TableCell>
                  <TableCell>
                    {student.first_name} {student.last_name}
                  </TableCell>
                  <TableCell>{student.email}</TableCell>
                  <TableCell>{student.faculty}</TableCell>
                  <TableCell>{student.department}</TableCell>
                  <TableCell>{student.enrollment_year}</TableCell>
                  <TableCell>{student.class_level}</TableCell>
                  <TableCell>
                    {student.advisor_name || (student as any).advisor ? (
                      <span className="text-sm">
                        {student.advisor_name || `${(student as any).advisor?.first_name} ${(student as any).advisor?.last_name}`}
                      </span>
                    ) : (
                      <span className="text-sm text-muted-foreground">No Advisor</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        student.status === 'active'
                          ? 'default'
                          : student.status === 'graduated'
                          ? 'secondary'
                          : 'destructive'
                      }
                    >
                      {student.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex space-x-2">
                      <Button variant="outline" size="sm" onClick={() => openEditModal(student)}>
                        Edit
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDeleteStudent(student.id)}
                      >
                        Delete
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <div className="flex justify-between items-center mt-4">
            <Button
              variant="outline"
              onClick={() => setCurrentPage((prev) => Math.max(prev - 1, 1))}
              disabled={currentPage === 1}
            >
              Previous
            </Button>
            <span>
              Page {currentPage} of {totalPages}
            </span>
            <Button
              variant="outline"
              onClick={() => setCurrentPage((prev) => Math.min(prev + 1, totalPages))}
              disabled={currentPage === totalPages}
            >
              Next
            </Button>
          </div>
        </>
      )}

      {/* Edit Dialog */}
      <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Student</DialogTitle>
            <DialogDescription>
              Update student information. Note: Only class level and status can be changed.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleUpdateStudent} className="space-y-4">
            <div>
              <Label htmlFor="edit_student_number">Student Number</Label>
              <Input
                id="edit_student_number"
                value={updateFormData.student_number}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_first_name">First Name</Label>
              <Input
                id="edit_first_name"
                value={updateFormData.first_name}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_last_name">Last Name</Label>
              <Input
                id="edit_last_name"
                value={updateFormData.last_name}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_email">Email</Label>
              <Input
                id="edit_email"
                type="email"
                value={updateFormData.email}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_faculty">Faculty</Label>
              <Input
                id="edit_faculty"
                value={updateFormData.faculty}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_department">Department</Label>
              <Input
                id="edit_department"
                value={updateFormData.department}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_enrollment_year">Enrollment Year</Label>
              <Input
                id="edit_enrollment_year"
                type="number"
                value={updateFormData.enrollment_year}
                disabled
              />
            </div>
            <div>
              <Label htmlFor="edit_class_level">Class Level</Label>
              <Input
                id="edit_class_level"
                type="number"
                min="1"
                max="6"
                value={updateFormData.class_level}
                onChange={(e) =>
                  setUpdateFormData({ ...updateFormData, class_level: parseInt(e.target.value) })
                }
                required
              />
            </div>
            <div>
              <Label htmlFor="edit_status">Status</Label>
              <select
                id="edit_status"
                value={updateFormData.status}
                onChange={(e) =>
                  setUpdateFormData({ ...updateFormData, status: e.target.value })
                }
                className="w-full border rounded-md p-2"
                required
              >
                <option value="active">Active</option>
                <option value="graduated">Graduated</option>
                <option value="suspended">Suspended</option>
                <option value="withdrawn">Withdrawn</option>
              </select>
            </div>
            <div>
              <Label htmlFor="edit_advisor">
                Danışman
                {loadingEditAdvisors && (
                  <Loader2 className="ml-2 h-4 w-4 animate-spin inline" />
                )}
              </Label>
              {loadingEditAdvisors ? (
                <div className="mt-1.5 p-2 border rounded-md text-sm text-muted-foreground">
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
                  {editDepartmentAdvisors.length} danışman bulundu ({updateFormData.department})
                </p>
              )}
            </div>
            <div className="flex justify-end space-x-2">
              <Button type="button" variant="outline" onClick={() => setIsEditOpen(false)}>
                Cancel
              </Button>
              <Button type="submit">Update</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
