import { useState, useEffect } from 'react';
import { Search, Loader2, ArrowLeft, Lock, Unlock, Calendar, GraduationCap } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { semesterApi } from '@/lib/api-client';
import { gradesService } from '@/lib/services/grades-service';
import type { CourseStatusResponse, StudentGrades, SemesterCourse } from '@/lib/types';
import { mockFaculties, mockCourseCatalog } from '@/mock_data/catalog';
import { mockAdminCourseStatus, mockAdminStudents } from '@/mock_data/admin_grades';
import { getActiveSemester } from '@/lib/services/system-service';

type AdminCourseRow = Pick<SemesterCourse, 'id' | 'course_code' | 'course_name' | 'department'> & {
  faculty?: string;
  instructor_fullname?: string;
};

type ViewState = 'COURSES' | 'STUDENTS';

export default function AdminGradesPage() {
  const [view, setView] = useState<ViewState>('COURSES');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Courses View State
  const [courses, setCourses] = useState<AdminCourseRow[]>([]);
  const [facultyFilter, setFacultyFilter] = useState('');
  const [departmentFilter, setDepartmentFilter] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [hasSearched, setHasSearched] = useState(false);

  // Selection State
  const [selectedCourse, setSelectedCourse] = useState<AdminCourseRow | null>(null);
  const [courseStatus, setCourseStatus] = useState<CourseStatusResponse | null>(null);
  
  // Students View State
  const [students, setStudents] = useState<StudentGrades[]>([]);
  const [lockDate, setLockDate] = useState<string>('');
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  
  type LockActionState = { student: StudentGrades; slug: string; isLocked: boolean; assessmentName?: string } | null;
  const [lockAction, setLockAction] = useState<LockActionState>(null);
  
  // Toggles testing functionality without backend
  const [useMockData, setUseMockData] = useState(false);

  // Active Semester State
  const [activeSemester, setActiveSemester] = useState<string>('');

  useEffect(() => {
    if (!useMockData) {
      getActiveSemester()
        .then((res) => {
          if (res) setActiveSemester(res.name);
        })
        .catch(console.error);
    } else {
      setActiveSemester('2025-2026 Güz');
    }
  }, [useMockData]);

  // Removed auto-fetching of courses on mount

  const handleFilterSearch = async () => {
    if (!facultyFilter || !departmentFilter) {
      setError('Lütfen önce fakülte ve bölüm seçimi yapınız.');
      return;
    }
    if (!useMockData && !activeSemester) {
      setError('Aktif dönem bulunamadı. Lütfen önce bir dönem aktifleştirin.');
      return;
    }
    setLoading(true);
    setHasSearched(true);
    setError('');
    try {
      if (useMockData) {
        const filtered = mockCourseCatalog.filter(c =>
          c.faculty === facultyFilter &&
          c.department === departmentFilter &&
          (searchQuery ? (c.course_code.toLowerCase().includes(searchQuery.toLowerCase()) || c.name.toLowerCase().includes(searchQuery.toLowerCase())) : true)
        );
        setCourses(filtered.map(c => ({
          id: c.id,
          course_code: c.course_code,
          course_name: c.name,
          department: c.department,
          faculty: c.faculty,
        })));
      } else {
        const resp = await semesterApi.get(`${encodeURIComponent(activeSemester)}/courses`, {
          searchParams: { faculty: facultyFilter, department: departmentFilter, limit: 100 },
        }).json<{ data: SemesterCourse[] | null }>();

        const q = searchQuery.trim().toLowerCase();
        const filtered = (resp.data || []).filter(c =>
          !q || c.course_code.toLowerCase().includes(q) || c.course_name.toLowerCase().includes(q)
        );
        setCourses(filtered.map(c => ({
          id: c.id,
          course_code: c.course_code,
          course_name: c.course_name,
          department: c.department,
          faculty: facultyFilter,
          instructor_fullname: c.instructor_fullname,
        })));
      }
    } catch (err: any) {
      if (err.response?.status === 404) {
        setCourses([]);
      } else {
        setError('Arama yapılırken hata oluştu.');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleCourseSelect = async (course: AdminCourseRow) => {
    setSelectedCourse(course);
    setView('STUDENTS');
    setLoading(true);
    try {
      if (useMockData) {
        setCourseStatus(mockAdminCourseStatus);
        setStudents(JSON.parse(JSON.stringify(mockAdminStudents))); // Deep clone to allow local modifications
      } else {
        const [statusRes, studentsRes] = await Promise.all([
          gradesService.getCourseStatus(course.id),
          gradesService.getCourseStudents(course.id)
        ]);
        setCourseStatus(statusRes);
        setStudents(studentsRes.students || []);
      }
    } catch (err) {
      setError('Ders verileri alınamadı.');
    } finally {
      setLoading(false);
    }
  };

  const toggleGradeLock = async (student: StudentGrades, slug: string, isLocked: boolean) => {
    setActionLoading(`${student.registration_id}-${slug}`);
    try {
      if (useMockData) {
        // Simulate local state update instead of API Call
        await new Promise(r => setTimeout(r, 400));
        setStudents(prev => prev.map(s => {
          if (s.registration_id === student.registration_id) {
            const updatedScores = { ...s.scores };
            if (updatedScores[slug]) {
              updatedScores[slug].is_locked = !isLocked;
            }
            return { ...s, scores: updatedScores };
          }
          return s;
        }));
      } else {
        if (isLocked) {
          await gradesService.unlockScore({ registration_id: student.registration_id, slug });
        } else {
          await gradesService.lockScore({ registration_id: student.registration_id, slug });
        }
        
        // Refresh students
        const studentsRes = await gradesService.getCourseStudents(selectedCourse!.id);
        setStudents(studentsRes.students || []);
      }
    } catch (err) {
      alert('İşlem başarısız oldu.');
    } finally {
      setActionLoading(null);
    }
  };

  // VIEWS 
  const renderCourses = () => {
    const selectedFaculty = mockFaculties.find((f) => f.name === facultyFilter);

    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
          <div className="grid flex-1 grid-cols-1 gap-4 sm:grid-cols-3">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Fakülte</label>
              <select
                className="flex h-10 w-full items-center justify-between rounded-md border border-gray-200 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-800 dark:bg-gray-950 dark:ring-offset-gray-950 dark:placeholder:text-gray-400 dark:focus:ring-gray-300"
                value={facultyFilter}
                onChange={(e) => {
                  setFacultyFilter(e.target.value);
                  setDepartmentFilter('');
                }}
              >
                <option value="">Fakülte Seçiniz</option>
                {mockFaculties.map((f) => (
                  <option key={f.id} value={f.name}>
                    {f.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Bölüm</label>
              <select
                className="flex h-10 w-full items-center justify-between rounded-md border border-gray-200 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-800 dark:bg-gray-950 dark:ring-offset-gray-950 dark:placeholder:text-gray-400 dark:focus:ring-gray-300"
                value={departmentFilter}
                onChange={(e) => setDepartmentFilter(e.target.value)}
                disabled={!selectedFaculty}
              >
                <option value="">Bölüm Seçiniz</option>
                {selectedFaculty?.departments.map((d) => (
                  <option key={d.id} value={d.name}>
                    {d.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Ders Ara</label>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                <Input
                  type="text"
                  placeholder="Kod veya İsim"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10 h-10"
                />
              </div>
            </div>
          </div>
          <Button onClick={handleFilterSearch} className="whitespace-nowrap h-10" disabled={!facultyFilter || !departmentFilter}>
            Filtrele
          </Button>
        </div>

        <div className="rounded-lg border bg-white dark:border-gray-800 dark:bg-gray-900">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Ders Kodu</TableHead>
                <TableHead>Ders Adı</TableHead>
                <TableHead>Fakülte / Bölüm</TableHead>
                <TableHead className="text-right">İşlemler</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={4} className="h-24 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-blue-600" />
                  </TableCell>
                </TableRow>
              ) : !hasSearched ? (
                <TableRow>
                  <TableCell colSpan={4} className="h-24 text-center font-medium text-gray-500">
                    Lütfen bir fakülte ve bölüm seçerek dersleri listeleyin.
                  </TableCell>
                </TableRow>
              ) : courses.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="h-24 text-center text-gray-500">
                    Ders bulunamadı.
                  </TableCell>
                </TableRow>
              ) : (
                courses.map(course => (
                  <TableRow key={course.id}>
                    <TableCell className="font-medium text-blue-600 dark:text-blue-400">
                      {course.course_code}
                    </TableCell>
                    <TableCell>{course.course_name}</TableCell>
                    <TableCell>
                      <div className="text-sm">
                        <div>{course.faculty || facultyFilter}</div>
                        <div className="text-gray-500">{course.department}</div>
                        {course.instructor_fullname && (
                          <div className="text-xs text-gray-400">{course.instructor_fullname}</div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="outline" size="sm" onClick={() => handleCourseSelect(course)}>
                        <Unlock className="mr-2 h-4 w-4" />
                        Notları Yönet
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    );
  };

  const renderStudents = () => {
    const assessments = courseStatus?.assessments || [];

    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="sm" onClick={() => setView('COURSES')}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Derslere Dön
            </Button>
            <div>
              <h2 className="text-xl font-bold text-gray-900 dark:text-white">
                {selectedCourse?.course_code} - Tüm Notlar
              </h2>
            </div>
          </div>
          
          <div className="flex items-center gap-2 rounded-lg border bg-white p-2 shadow-sm dark:border-gray-800 dark:bg-gray-900">
            <Calendar className="h-5 w-5 text-gray-500" />
            <div className="flex flex-col">
              <span className="text-xs font-semibold text-gray-500 uppercase">Kilitlenme Tarihi Belirle</span>
              <Input 
                type="datetime-local" 
                className="h-8 border-none p-0 text-sm shadow-none focus-visible:ring-0"
                value={lockDate}
                onChange={(e) => setLockDate(e.target.value)}
              />
            </div>
            {lockDate && (
               <Button size="sm" onClick={() => alert('Frontend kilit tarihi kaydedildi: ' + lockDate)}>Kaydet</Button>
            )}
          </div>
        </div>

        <div className="rounded-lg border bg-white dark:border-gray-800 dark:bg-gray-900 overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="whitespace-nowrap">Öğrenci No</TableHead>
                <TableHead className="whitespace-nowrap">Ad Soyad</TableHead>
                {assessments.map(a => (
                  <TableHead key={a.slug} className="text-center min-w-[140px] whitespace-nowrap">
                    {a.name}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={2 + assessments.length} className="h-24 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-blue-600" />
                  </TableCell>
                </TableRow>
              ) : students.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={2 + assessments.length} className="h-24 text-center text-gray-500">
                    Öğrenci bulunamadı.
                  </TableCell>
                </TableRow>
              ) : (
                students.map(student => (
                  <TableRow key={student.registration_id}>
                    <TableCell className="font-mono text-sm">{student.student_number}</TableCell>
                    <TableCell className="whitespace-nowrap">{student.first_name} {student.last_name}</TableCell>
                    {assessments.map(a => {
                      const existingScore = student.scores[a.slug];
                      const isLocked = existingScore?.is_locked ?? false;
                      const isLoading = actionLoading === `${student.registration_id}-${a.slug}`;
                      
                      return (
                        <TableCell key={a.slug} className="text-center align-top py-4">
                          <div className="flex flex-col items-center justify-center gap-2">
                            <span className="font-bold text-lg">
                              {existingScore ? (existingScore.is_absent ? 'D.' : existingScore.score) : '-'}
                            </span>
                            {existingScore && (
                              <Button 
                                variant={isLocked ? 'default' : 'destructive'} 
                                size="sm" 
                                className="h-8 text-xs w-full max-w-[100px]"
                                disabled={isLoading}
                                onClick={() => setLockAction({ student, slug: a.slug, isLocked, assessmentName: a.name })}
                              >
                                {isLoading ? (
                                  <Loader2 className="h-3 w-3 animate-spin"/>
                                ) : isLocked ? (
                                  <><Unlock className="mr-1 h-3 w-3"/> Kilidi Aç</>
                                ) : (
                                  <><Lock className="mr-1 h-3 w-3"/> Kilitle</>
                                )}
                              </Button>
                            )}
                            {!existingScore && (
                              <span className="text-xs text-gray-400">Not Girilmemiş</span>
                            )}
                          </div>
                        </TableCell>
                      );
                    })}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Admin Not Yönetimi</h1>
            
            {activeSemester && (
              <div className="mt-2 inline-flex items-center gap-1.5 rounded-md bg-blue-50 px-2 py-1 text-xs font-semibold text-blue-700 ring-1 ring-inset ring-blue-700/10 dark:bg-blue-900/30 dark:text-blue-300 dark:ring-blue-900/50">
                <GraduationCap className="h-3.5 w-3.5" />
                Aktif Dönem: {activeSemester}
              </div>
            )}

            <p className="mt-2 text-sm text-gray-500">
              Öğrenci notları ve değerlendirmelerin kilit durumlarını buradan öğrenci bazında yönetebilirsiniz.
            </p>
          </div>
          <div className="flex items-center gap-2 rounded border px-3 py-1.5 shadow-sm dark:border-gray-800 bg-white dark:bg-gray-900">
             <label htmlFor="mock-toggle" className="text-xs font-semibold text-gray-700 dark:text-gray-300 select-none cursor-pointer">
               Test Modu (Mock Veri)
             </label>
             <input 
               id="mock-toggle" 
               type="checkbox" 
               className="cursor-pointer rounded accent-blue-600"
               checked={useMockData} 
               onChange={(e) => {
                 setUseMockData(e.target.checked);
                 setHasSearched(false);
                 setCourses([]);
                 if (view === 'STUDENTS') setView('COURSES');
               }} 
             />
          </div>
        </div>
      </div>

      {error && (
        <div className="rounded-md bg-red-50 p-4 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400">
          {error}
        </div>
      )}

      {view === 'COURSES' && renderCourses()}
      {view === 'STUDENTS' && renderStudents()}

      <AlertDialog open={!!lockAction} onOpenChange={(open) => !open && setLockAction(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Notu {lockAction?.isLocked ? 'Kilidini Aç' : 'Kilitle'}</AlertDialogTitle>
            <AlertDialogDescription>
              <strong>{lockAction?.student.first_name} {lockAction?.student.last_name}</strong> isimli öğrencinin <strong>{lockAction?.assessmentName}</strong> notunu {lockAction?.isLocked ? 'açmak' : 'kilitlemek'} istediğinize emin misiniz?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>İptal</AlertDialogCancel>
            <AlertDialogAction onClick={() => {
              if (lockAction) {
                toggleGradeLock(lockAction.student, lockAction.slug, lockAction.isLocked);
                setLockAction(null);
              }
            }}>
              Evet, Onayla
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
