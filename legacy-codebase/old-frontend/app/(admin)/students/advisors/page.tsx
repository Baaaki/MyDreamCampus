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
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { studentApi, staffApi } from '@/lib/api-client'
import { ArrowLeft, UserPlus, Users, ArrowUpDown, ArrowUp, ArrowDown, Loader2 } from 'lucide-react'
import Link from 'next/link'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost'

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
  advisor?: {
    id: string
    first_name: string
    last_name: string
    email?: string
  }
  status: string
  created_at: string
  updated_at: string
}

type Staff = {
  id: string
  first_name: string
  last_name: string
  email: string
  title?: string
  faculty?: string
  department: string
  office_location?: string
  role?: string
  status?: string
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

type StaffListResponse = {
  data: Staff[]
  pagination: {
    page: number
    limit: number
    total: number
    total_pages: number
  }
}

export default function AdvisorManagementPage() {
  const [orphanedStudents, setOrphanedStudents] = useState<Student[]>([])
  const [advisorStudents, setAdvisorStudents] = useState<Student[]>([])
  const [staffList, setStaffList] = useState<Staff[]>([])
  const [selectedAdvisor, setSelectedAdvisor] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [isAssignOpen, setIsAssignOpen] = useState(false)
  const [isBulkAssignOpen, setIsBulkAssignOpen] = useState(false)
  const [assigningStudent, setAssigningStudent] = useState<Student | null>(null)
  const [selectedAdvisorId, setSelectedAdvisorId] = useState<string>('')
  const [bulkAdvisorId, setBulkAdvisorId] = useState<string>('')
  const [selectedStudentIds, setSelectedStudentIds] = useState<string[]>([])

  // Cascade dropdown states for bulk assignment
  const [bulkSelectedFaculty, setBulkSelectedFaculty] = useState<string>('')
  const [bulkSelectedDepartment, setBulkSelectedDepartment] = useState<string>('')

  // Department-specific instructors (fetched from API)
  const [departmentInstructors, setDepartmentInstructors] = useState<Staff[]>([])
  const [loadingInstructors, setLoadingInstructors] = useState(false)

  // Pagination states
  const [orphanedPage, setOrphanedPage] = useState(1)
  const [orphanedTotalPages, setOrphanedTotalPages] = useState(1)
  const [orphanedTotal, setOrphanedTotal] = useState(0)
  const [advisorPage, setAdvisorPage] = useState(1)
  const [advisorTotalPages, setAdvisorTotalPages] = useState(1)
  const [advisorTotal, setAdvisorTotal] = useState(0)
  const [limit] = useState(10)

  // Sort states for orphaned students
  const [orphanedSortField, setOrphanedSortField] = useState<keyof Student>('student_number')
  const [orphanedSortDirection, setOrphanedSortDirection] = useState<'asc' | 'desc'>('asc')

  // Sort states for advisor students
  const [advisorSortField, setAdvisorSortField] = useState<keyof Student>('student_number')
  const [advisorSortDirection, setAdvisorSortDirection] = useState<'asc' | 'desc'>('asc')

  useEffect(() => {
    fetchStaffList()
    fetchOrphanedStudents()
  }, [orphanedPage])

  useEffect(() => {
    if (selectedAdvisor) {
      fetchAdvisorStudents()
    }
  }, [selectedAdvisor, advisorPage])

  const fetchStaffList = async () => {
    try {
      const response = (await staffApi
        .get('', {
          searchParams: {
            page: '1',
            limit: '100',
          },
        })
        .json()) as StaffListResponse

      setStaffList(response.data)
      console.log('Staff list loaded:', response.data.length, 'staff members')
    } catch (error) {
      console.error('Failed to fetch staff list:', error)
    }
  }

  const fetchOrphanedStudents = async () => {
    setLoading(true)
    try {
      const response = (await studentApi
        .get('orphaned', {
          searchParams: {
            page: orphanedPage.toString(),
            limit: limit.toString(),
          },
        })
        .json()) as StudentListResponse

      if (response && response.data) {
        setOrphanedStudents(response.data)
        setOrphanedTotalPages(response.pagination.total_pages)
        setOrphanedTotal(response.pagination.total)
        console.log('Orphaned students loaded:', response.data.length, 'students')
      } else {
        console.warn('No data received for orphaned students')
        setOrphanedStudents([])
        setOrphanedTotalPages(1)
        setOrphanedTotal(0)
      }
    } catch (error) {
      console.error('Failed to fetch orphaned students:', error)
      setOrphanedStudents([])
      setOrphanedTotalPages(1)
      setOrphanedTotal(0)
    } finally {
      setLoading(false)
    }
  }

  const fetchAdvisorStudents = async () => {
    setLoading(true)
    try {
      const response = (await studentApi
        .get(`advisors/${selectedAdvisor}/advisees`, {
          searchParams: {
            page: advisorPage.toString(),
            limit: limit.toString(),
          },
        })
        .json()) as StudentListResponse

      if (response && response.data) {
        setAdvisorStudents(response.data)
        setAdvisorTotalPages(response.pagination.total_pages)
        setAdvisorTotal(response.pagination.total)
        console.log('Advisor students loaded:', response.data.length, 'students')
      } else {
        console.warn('No data received for advisor students')
        setAdvisorStudents([])
        setAdvisorTotalPages(1)
        setAdvisorTotal(0)
      }
    } catch (error) {
      console.error('Failed to fetch advisor students:', error)
      setAdvisorStudents([])
      setAdvisorTotalPages(1)
      setAdvisorTotal(0)
    } finally {
      setLoading(false)
    }
  }

  const handleAssignAdvisor = async (e: React.FormEvent) => {
    e.preventDefault()
    console.log('[Advisors] handleAssignAdvisor called', { assigningStudent, selectedAdvisorId })
    if (!assigningStudent || !selectedAdvisorId) {
      console.log('[Advisors] Missing data, returning early')
      return
    }

    try {
      console.log('[Advisors] Sending PUT request to:', `${assigningStudent.id}`)
      const response = await studentApi.put(`${assigningStudent.id}`, {
        json: {
          advisor_id: selectedAdvisorId,
        },
      }).json()
      console.log('[Advisors] PUT response:', response)

      setIsAssignOpen(false)
      setAssigningStudent(null)
      setSelectedAdvisorId('')
      fetchOrphanedStudents()
      if (selectedAdvisor) {
        fetchAdvisorStudents()
      }
    } catch (error) {
      console.error('Failed to assign advisor:', error)
    }
  }

  const handleBulkAssign = async (e: React.FormEvent) => {
    e.preventDefault()
    if (selectedStudentIds.length === 0 || !bulkAdvisorId) return

    try {
      await studentApi.post('advisors/bulk-assign', {
        json: {
          student_ids: selectedStudentIds,
          advisor_id: bulkAdvisorId,
        },
      })

      setIsBulkAssignOpen(false)
      setSelectedStudentIds([])
      setBulkSelectedFaculty('')
      setBulkSelectedDepartment('')
      setBulkAdvisorId('')
      fetchOrphanedStudents()
      if (selectedAdvisor) {
        fetchAdvisorStudents()
      }
    } catch (error) {
      console.error('Failed to bulk assign advisor:', error)
    }
  }

  const toggleStudentSelection = (studentId: string) => {
    setSelectedStudentIds((prev) =>
      prev.includes(studentId)
        ? prev.filter((id) => id !== studentId)
        : [...prev, studentId]
    )
  }

  // Sort handler for orphaned students
  const handleOrphanedSort = (field: keyof Student) => {
    if (orphanedSortField === field) {
      setOrphanedSortDirection(orphanedSortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setOrphanedSortField(field)
      setOrphanedSortDirection('asc')
    }
  }

  // Sort handler for advisor students
  const handleAdvisorSort = (field: keyof Student) => {
    if (advisorSortField === field) {
      setAdvisorSortDirection(advisorSortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setAdvisorSortField(field)
      setAdvisorSortDirection('asc')
    }
  }

  // Sort icon component
  const SortIcon = ({ field, currentField, direction }: { field: string; currentField: string; direction: 'asc' | 'desc' }) => {
    if (field !== currentField) {
      return <ArrowUpDown className="ml-2 h-4 w-4 inline" />
    }
    return direction === 'asc' ? (
      <ArrowUp className="ml-2 h-4 w-4 inline" />
    ) : (
      <ArrowDown className="ml-2 h-4 w-4 inline" />
    )
  }

  // Apply sorting to orphaned students
  const sortedOrphanedStudents = [...orphanedStudents].sort((a, b) => {
    const aValue = a[orphanedSortField]
    const bValue = b[orphanedSortField]

    if (aValue === null || aValue === undefined) return 1
    if (bValue === null || bValue === undefined) return -1

    if (typeof aValue === 'string' && typeof bValue === 'string') {
      return orphanedSortDirection === 'asc'
        ? aValue.localeCompare(bValue)
        : bValue.localeCompare(aValue)
    }

    if (typeof aValue === 'number' && typeof bValue === 'number') {
      return orphanedSortDirection === 'asc' ? aValue - bValue : bValue - aValue
    }

    return 0
  })

  // Apply sorting to advisor students
  const sortedAdvisorStudents = [...advisorStudents].sort((a, b) => {
    const aValue = a[advisorSortField]
    const bValue = b[advisorSortField]

    if (aValue === null || aValue === undefined) return 1
    if (bValue === null || bValue === undefined) return -1

    if (typeof aValue === 'string' && typeof bValue === 'string') {
      return advisorSortDirection === 'asc'
        ? aValue.localeCompare(bValue)
        : bValue.localeCompare(aValue)
    }

    if (typeof aValue === 'number' && typeof bValue === 'number') {
      return advisorSortDirection === 'asc' ? aValue - bValue : bValue - aValue
    }

    return 0
  })

  // Helper: Get unique departments from staff list (used as "faculties" since backend doesn't have faculty field)
  const getUniqueFaculties = () => {
    const departments = new Set<string>()
    staffList.forEach((staff) => {
      if (staff.department) {
        departments.add(staff.department)
      }
    })
    const result = Array.from(departments).sort()
    console.log('Unique Departments (as faculties):', result, 'Staff List Length:', staffList.length)
    return result
  }

  // Helper: Get departments for a specific faculty (returns same department since we're using department as faculty)
  const getDepartmentsForFaculty = (faculty: string) => {
    // Since backend doesn't have faculty, we treat department as both faculty and department
    // Return the same department if it exists in staff list
    const hasStaffInDepartment = staffList.some((staff) => staff.department === faculty)
    return hasStaffInDepartment ? [faculty] : []
  }

  // Fetch instructors by department from API (with Turkish-English normalization on backend)
  const fetchInstructorsByDepartment = async (department: string): Promise<Staff[]> => {
    try {
      const response = await fetch(
        `${API_BASE_URL}:8002/internal/staff/instructors?department=${encodeURIComponent(department)}`
      )

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const data = await response.json()
      const instructors = data.data || []
      console.log(`[Advisors] Fetched ${instructors.length} instructors for department: ${department}`)
      return instructors
    } catch (err) {
      console.error('Error fetching instructors by department:', err)
      // Fallback to client-side filtering
      return staffList.filter((staff) => staff.department === department)
    }
  }

  // Helper: Get advisors for a specific department (uses API with caching)
  const getAdvisorsForDepartment = (department: string) => {
    // If we have department-specific instructors loaded, use them
    if (departmentInstructors.length > 0 && assigningStudent?.department === department) {
      return departmentInstructors
    }
    // Fallback to all staff if no filter or loading
    if (!department) return staffList
    return staffList
  }

  // Load instructors when assigning student changes
  useEffect(() => {
    if (assigningStudent?.department) {
      setLoadingInstructors(true)
      fetchInstructorsByDepartment(assigningStudent.department)
        .then((instructors) => {
          setDepartmentInstructors(instructors) // Bölümde hoca yoksa boş kalacak
        })
        .finally(() => {
          setLoadingInstructors(false)
        })
    } else {
      setDepartmentInstructors([])
    }
  }, [assigningStudent])



  return (
    <div className="container mx-auto py-10">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <Link href="/students">
            <Button variant="ghost" size="icon">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </Link>
          <h1 className="text-3xl font-bold">Danışman İşleri</h1>
        </div>
      </div>

      {/* Orphaned Students Section */}
      <Card className="mb-6">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Danışmanı Olmayan Öğrenciler</CardTitle>
              <CardDescription>
                Bu öğrencilere danışman atayabilirsiniz
              </CardDescription>
            </div>
            <Dialog open={isBulkAssignOpen} onOpenChange={(open) => {
              setIsBulkAssignOpen(open)
              if (!open) {
                setBulkSelectedFaculty('')
                setBulkSelectedDepartment('')
                setBulkAdvisorId('')
              }
            }}>
              <DialogTrigger asChild>
                <Button
                  disabled={selectedStudentIds.length === 0}
                  onClick={() => {
                    setBulkSelectedFaculty('')
                    setBulkSelectedDepartment('')
                    setBulkAdvisorId('')
                  }}
                >
                  <Users className="mr-2 h-4 w-4" />
                  Toplu Atama ({selectedStudentIds.length})
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Toplu Danışman Atama</DialogTitle>
                  <DialogDescription>
                    Seçili {selectedStudentIds.length} öğrenciye danışman atayın
                  </DialogDescription>
                </DialogHeader>
                <form onSubmit={handleBulkAssign} className="space-y-4">
                  <div>
                    <Label htmlFor="bulk_faculty">Fakülte</Label>
                    <Select
                      value={bulkSelectedFaculty}
                      onValueChange={(value) => {
                        setBulkSelectedFaculty(value)
                        setBulkSelectedDepartment('')
                        setBulkAdvisorId('')
                      }}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Fakülte seçin" />
                      </SelectTrigger>
                      <SelectContent>
                        {getUniqueFaculties().map((faculty) => (
                          <SelectItem key={faculty} value={faculty}>
                            {faculty}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {bulkSelectedFaculty && (
                    <div>
                      <Label htmlFor="bulk_department">Bölüm</Label>
                      <Select
                        value={bulkSelectedDepartment}
                        onValueChange={(value) => {
                          setBulkSelectedDepartment(value)
                          setBulkAdvisorId('')
                        }}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Bölüm seçin" />
                        </SelectTrigger>
                        <SelectContent>
                          {getDepartmentsForFaculty(bulkSelectedFaculty).map((dept) => (
                            <SelectItem key={dept} value={dept}>
                              {dept}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}

                  {bulkSelectedDepartment && (
                    <div>
                      <Label htmlFor="bulk_advisor">Danışman</Label>
                      <Select value={bulkAdvisorId} onValueChange={setBulkAdvisorId}>
                        <SelectTrigger>
                          <SelectValue placeholder="Danışman seçin" />
                        </SelectTrigger>
                        <SelectContent>
                          {getAdvisorsForDepartment(bulkSelectedDepartment).map((staff) => (
                            <SelectItem key={staff.id} value={staff.id}>
                              {staff.first_name} {staff.last_name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}

                  <div className="flex justify-end space-x-2">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => {
                        setIsBulkAssignOpen(false)
                        setBulkSelectedFaculty('')
                        setBulkSelectedDepartment('')
                        setBulkAdvisorId('')
                      }}
                    >
                      İptal
                    </Button>
                    <Button type="submit" disabled={!bulkAdvisorId}>
                      Toplu Atama Yap
                    </Button>
                  </div>
                </form>
              </DialogContent>
            </Dialog>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-center py-4">Yükleniyor...</div>
          ) : orphanedStudents.length === 0 ? (
            <div className="text-center py-4 text-muted-foreground">
              Danışmanı olmayan öğrenci bulunmamaktadır.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[50px]">
                    <input
                      type="checkbox"
                      checked={selectedStudentIds.length === orphanedStudents.length}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setSelectedStudentIds(orphanedStudents.map((s) => s.id))
                        } else {
                          setSelectedStudentIds([])
                        }
                      }}
                      className="cursor-pointer"
                    />
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none hover:bg-muted/50"
                    onClick={() => handleOrphanedSort('student_number')}
                  >
                    Öğrenci No
                    <SortIcon field="student_number" currentField={orphanedSortField} direction={orphanedSortDirection} />
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none hover:bg-muted/50"
                    onClick={() => handleOrphanedSort('first_name')}
                  >
                    Ad Soyad
                    <SortIcon field="first_name" currentField={orphanedSortField} direction={orphanedSortDirection} />
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none hover:bg-muted/50"
                    onClick={() => handleOrphanedSort('email')}
                  >
                    Email
                    <SortIcon field="email" currentField={orphanedSortField} direction={orphanedSortDirection} />
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none hover:bg-muted/50"
                    onClick={() => handleOrphanedSort('faculty')}
                  >
                    Fakülte
                    <SortIcon field="faculty" currentField={orphanedSortField} direction={orphanedSortDirection} />
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none hover:bg-muted/50"
                    onClick={() => handleOrphanedSort('department')}
                  >
                    Bölüm
                    <SortIcon field="department" currentField={orphanedSortField} direction={orphanedSortDirection} />
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none hover:bg-muted/50"
                    onClick={() => handleOrphanedSort('class_level')}
                  >
                    Sınıf
                    <SortIcon field="class_level" currentField={orphanedSortField} direction={orphanedSortDirection} />
                  </TableHead>
                  <TableHead>İşlemler</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sortedOrphanedStudents.map((student) => (
                  <TableRow key={student.id}>
                    <TableCell>
                      <input
                        type="checkbox"
                        checked={selectedStudentIds.includes(student.id)}
                        onChange={() => toggleStudentSelection(student.id)}
                        className="cursor-pointer"
                      />
                    </TableCell>
                    <TableCell className="font-medium">{student.student_number}</TableCell>
                    <TableCell>
                      {student.first_name} {student.last_name}
                    </TableCell>
                    <TableCell>{student.email}</TableCell>
                    <TableCell>{student.faculty}</TableCell>
                    <TableCell>{student.department}</TableCell>
                    <TableCell>{student.class_level}</TableCell>
                    <TableCell>
                      <Dialog open={isAssignOpen && assigningStudent?.id === student.id} onOpenChange={(open) => {
                        setIsAssignOpen(open)
                        if (!open) {
                          setAssigningStudent(null)
                          setSelectedAdvisorId('')
                        }
                      }}>
                        <DialogTrigger asChild>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => {
                              setAssigningStudent(student)
                              setSelectedAdvisorId('')
                            }}
                          >
                            <UserPlus className="mr-2 h-4 w-4" />
                            Danışman Ata
                          </Button>
                        </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>Danışman Atama</DialogTitle>
                            <DialogDescription>
                              {student.first_name} {student.last_name} için danışman seçin
                            </DialogDescription>
                          </DialogHeader>
                            <form onSubmit={handleAssignAdvisor} className="space-y-4">
                              <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-1">
                                  <Label className="text-muted-foreground text-xs">Fakülte</Label>
                                  <div className="font-medium text-sm">{student.faculty || '-'}</div>
                                </div>
                                <div className="space-y-1">
                                  <Label className="text-muted-foreground text-xs">Bölüm</Label>
                                  <div className="font-medium text-sm">{student.department || '-'}</div>
                                </div>
                              </div>

                              <div>
                                <Label htmlFor="advisor">
                                  Danışman Seçin
                                  {loadingInstructors && (
                                    <Loader2 className="ml-2 h-4 w-4 animate-spin inline" />
                                  )}
                                </Label>
                                {loadingInstructors ? (
                                  <div className="mt-1.5 p-2 border rounded-md text-sm text-muted-foreground">
                                    Danışmanlar yükleniyor...
                                  </div>
                                ) : (
                                  <>
                                    <Select value={selectedAdvisorId} onValueChange={setSelectedAdvisorId}>
                                      <SelectTrigger className="mt-1.5">
                                        <SelectValue placeholder="Danışman seçin" />
                                      </SelectTrigger>
                                      <SelectContent>
                                        {departmentInstructors.map((staff) => (
                                          <SelectItem key={staff.id} value={staff.id}>
                                            {staff.first_name} {staff.last_name} {staff.department !== student.department ? `(${staff.department})` : ''}
                                          </SelectItem>
                                        ))}
                                      </SelectContent>
                                    </Select>
                                    {departmentInstructors.length > 0 && (
                                      <p className="mt-1 text-xs text-green-600">
                                        {departmentInstructors.length} danışman bulundu
                                      </p>
                                    )}
                                  </>
                                )}
                              </div>

                            <div className="flex justify-end space-x-2">
                              <Button
                                type="button"
                                variant="outline"
                                onClick={() => {
                                  setIsAssignOpen(false)
                                  setSelectedAdvisorId('')
                                }}
                              >
                                İptal
                              </Button>
                              <Button type="submit" disabled={!selectedAdvisorId}>
                                Danışman Ata
                              </Button>
                            </div>
                          </form>
                        </DialogContent>
                      </Dialog>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}

          {/* Pagination for Orphaned Students */}
          {!loading && orphanedStudents.length > 0 && (
            <div className="flex items-center justify-between mt-4">
              <div className="text-sm text-muted-foreground">
                Showing {orphanedStudents.length} of {orphanedTotal} students
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setOrphanedPage((prev) => Math.max(1, prev - 1))}
                  disabled={orphanedPage === 1}
                >
                  Previous
                </Button>
                <div className="flex items-center gap-2">
                  <span className="text-sm">
                    Page {orphanedPage} of {orphanedTotalPages}
                  </span>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setOrphanedPage((prev) => Math.min(orphanedTotalPages, prev + 1))}
                  disabled={orphanedPage === orphanedTotalPages}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Advisor's Students Section */}
      <Card>
        <CardHeader>
          <CardTitle>Danışmanın Öğrencileri</CardTitle>
          <CardDescription>Bir danışman seçerek öğrencilerini görün</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="mb-4">
            <Label htmlFor="advisor_select">Danışman Seçin</Label>
            <Select
              value={selectedAdvisor}
              onValueChange={(value) => {
                setSelectedAdvisor(value)
                setAdvisorPage(1) // Reset to page 1 when changing advisor
              }}
            >
              <SelectTrigger>
                <SelectValue placeholder="Danışman seçin" />
              </SelectTrigger>
              <SelectContent>
                {staffList.map((staff) => (
                  <SelectItem key={staff.id} value={staff.id}>
                    {staff.first_name} {staff.last_name} - {staff.department}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {selectedAdvisor && (
            <>
              {loading ? (
                <div className="text-center py-4">Yükleniyor...</div>
              ) : advisorStudents.length === 0 ? (
                <div className="text-center py-4 text-muted-foreground">
                  Bu danışmanın öğrencisi bulunmamaktadır.
                </div>
              ) : (
                <>
                  <Table>
                    <TableCaption>
                      Toplam {advisorTotal} öğrenci
                    </TableCaption>
                    <TableHeader>
                      <TableRow>
                        <TableHead
                          className="cursor-pointer select-none hover:bg-muted/50"
                          onClick={() => handleAdvisorSort('student_number')}
                        >
                          Öğrenci No
                          <SortIcon field="student_number" currentField={advisorSortField} direction={advisorSortDirection} />
                        </TableHead>
                        <TableHead
                          className="cursor-pointer select-none hover:bg-muted/50"
                          onClick={() => handleAdvisorSort('first_name')}
                        >
                          Ad Soyad
                          <SortIcon field="first_name" currentField={advisorSortField} direction={advisorSortDirection} />
                        </TableHead>
                        <TableHead
                          className="cursor-pointer select-none hover:bg-muted/50"
                          onClick={() => handleAdvisorSort('email')}
                        >
                          Email
                          <SortIcon field="email" currentField={advisorSortField} direction={advisorSortDirection} />
                        </TableHead>
                        <TableHead
                          className="cursor-pointer select-none hover:bg-muted/50"
                          onClick={() => handleAdvisorSort('faculty')}
                        >
                          Fakülte
                          <SortIcon field="faculty" currentField={advisorSortField} direction={advisorSortDirection} />
                        </TableHead>
                        <TableHead
                          className="cursor-pointer select-none hover:bg-muted/50"
                          onClick={() => handleAdvisorSort('department')}
                        >
                          Bölüm
                          <SortIcon field="department" currentField={advisorSortField} direction={advisorSortDirection} />
                        </TableHead>
                        <TableHead
                          className="cursor-pointer select-none hover:bg-muted/50"
                          onClick={() => handleAdvisorSort('enrollment_year')}
                        >
                          Kayıt Yılı
                          <SortIcon field="enrollment_year" currentField={advisorSortField} direction={advisorSortDirection} />
                        </TableHead>
                        <TableHead
                          className="cursor-pointer select-none hover:bg-muted/50"
                          onClick={() => handleAdvisorSort('class_level')}
                        >
                          Sınıf
                          <SortIcon field="class_level" currentField={advisorSortField} direction={advisorSortDirection} />
                        </TableHead>
                        <TableHead
                          className="cursor-pointer select-none hover:bg-muted/50"
                          onClick={() => handleAdvisorSort('status')}
                        >
                          Durum
                          <SortIcon field="status" currentField={advisorSortField} direction={advisorSortDirection} />
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {sortedAdvisorStudents.map((student) => (
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
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>

                  {/* Pagination for Advisor Students */}
                  <div className="flex items-center justify-between mt-4">
                    <div className="text-sm text-muted-foreground">
                      Showing {advisorStudents.length} of {advisorTotal} students
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setAdvisorPage((prev) => Math.max(1, prev - 1))}
                        disabled={advisorPage === 1}
                      >
                        Previous
                      </Button>
                      <div className="flex items-center gap-2">
                        <span className="text-sm">
                          Page {advisorPage} of {advisorTotalPages}
                        </span>
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setAdvisorPage((prev) => Math.min(advisorTotalPages, prev + 1))}
                        disabled={advisorPage === advisorTotalPages}
                      >
                        Next
                      </Button>
                    </div>
                  </div>
                </>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
