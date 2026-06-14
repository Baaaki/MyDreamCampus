
import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { mockFaculties } from '@/mock_data/catalog';
import type { Faculty, Department, AssessmentItem, ScheduleSessionDTO, CourseCatalog, CreateSemesterCourseRequest, SemesterCourse } from '@/lib/types';
import { catalogApi, staffApi, semesterApi } from '@/lib/api-client';
import {
  Plus,
  Trash2,
  CalendarPlus,
  GraduationCap,
  User,
  MapPin,
  Users,
  Clock,
  Percent,
  Save,
  AlertCircle,
  CheckCircle2,
  BookOpen,
  Loader2,
} from 'lucide-react';

// Types for API responses
interface Instructor {
  id: string;
  fullname: string;
  title: string;
  first_name?: string;
  last_name?: string;
}

interface CoursesResponse {
  data: CourseCatalog[];
  pagination: {
    total: number;
    page: number;
    limit: number;
    total_pages: number;
  };
}

interface StaffResponse {
  data: Array<{
    id: string;
    first_name: string;
    last_name: string;
    role: string;
    faculty?: string;
    department?: string;
  }>;
  pagination: {
    total: number;
    page: number;
    limit: number;
    total_pages: number;
  };
}

// Fetch courses from catalog API
const fetchCourses = async (department: string): Promise<CourseCatalog[]> => {
  const response = await catalogApi.get('courses', {
    searchParams: { department, limit: 100 },
  }).json<CoursesResponse>();
  return response.data || [];
};

// Fetch instructors from staff API
const fetchInstructors = async (department: string): Promise<Instructor[]> => {
  const response = await staffApi.get('instructors', {
    searchParams: { department, limit: 100 },
  }).json<StaffResponse>();

  return (response.data || []).map(staff => ({
    id: staff.id,
    fullname: `${staff.first_name} ${staff.last_name}`,
    title: staff.role,
    first_name: staff.first_name,
    last_name: staff.last_name,
  }));
};

// Create semester course via API
const createSemesterCourse = async (semester: string, data: CreateSemesterCourseRequest) => {
  return semesterApi.post(`${semester}/courses`, { json: data }).json();
};

// Fetch existing semester courses for a department
const fetchSemesterCourses = async (semester: string, department: string) => {
  const response = await semesterApi.get(`${semester}/courses`, {
    searchParams: { department, limit: 100 },
  }).json<{ data: SemesterCourse[] }>();
  return response.data || [];
};

// Predefined assessment types
const predefinedAssessments = [
  { slug: 'midterm', name: 'Vize' },
  { slug: 'final', name: 'Final' },
  { slug: 'quiz', name: 'Quiz' },
  { slug: 'homework', name: 'Ödev' },
  { slug: 'project', name: 'Proje' },
  { slug: 'lab', name: 'Laboratuvar' },
  { slug: 'presentation', name: 'Sunum' },
  { slug: 'attendance', name: 'Devam' },
];

// Days of week
const daysOfWeek = [
  { key: 'monday', label: 'Pazartesi' },
  { key: 'tuesday', label: 'Salı' },
  { key: 'wednesday', label: 'Çarşamba' },
  { key: 'thursday', label: 'Perşembe' },
  { key: 'friday', label: 'Cuma' },
];

// Time slots (ders saatleri)
const timeSlots = [
  { slot: 1, time: '08:30 - 09:15' },
  { slot: 2, time: '09:25 - 10:10' },
  { slot: 3, time: '10:20 - 11:05' },
  { slot: 4, time: '11:15 - 12:00' },
  { slot: 5, time: '12:10 - 12:55' },
  { slot: 6, time: '13:00 - 13:45' },
  { slot: 7, time: '13:55 - 14:40' },
  { slot: 8, time: '14:50 - 15:35' },
  { slot: 9, time: '15:45 - 16:30' },
];

interface FormData {
  faculty_id: string;
  department_id: string;
  course_id: string;
  class_level: number;
  instructor_id: string;
  instructor_fullname: string;
  classroom_location: string;
  max_capacity: number;
  assessment_schema: AssessmentItem[];
  schedule_sessions: ScheduleSessionDTO[];
}

const initialFormData: FormData = {
  faculty_id: '',
  department_id: '',
  course_id: '',
  class_level: 1,
  instructor_id: '',
  instructor_fullname: '',
  classroom_location: '',
  max_capacity: 50,
  assessment_schema: [
    { slug: 'midterm', name: 'Vize', weight: 40 },
    { slug: 'final', name: 'Final', weight: 60 },
  ],
  schedule_sessions: [],
};

export default function SemesterCoursesPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [formData, setFormData] = useState<FormData>(initialFormData);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [selectedCourse, setSelectedCourse] = useState<CourseCatalog | null>(null);
  const [semester, setSemester] = useState('2024-2025-Fall'); // Current semester
  const [activeSessionType, setActiveSessionType] = useState<'theory' | 'lab'>('theory');
  const [errorDialog, setErrorDialog] = useState<{ open: boolean; title: string; message: string }>({ open: false, title: '', message: '' });
  const [successDialog, setSuccessDialog] = useState(false);

  // Calculate total assessment weight
  const totalWeight = formData.assessment_schema.reduce((sum, item) => sum + item.weight, 0);
  const isWeightValid = totalWeight === 100;

  // Get department name from ID for API calls
  const getDepartmentName = (deptId: string): string => {
    const dept = departments.find(d => d.id === deptId);
    return dept?.name || '';
  };

  // TanStack Query - Fetch courses from catalog API
  const {
    data: courses = [],
    isLoading: isLoadingCourses,
  } = useQuery({
    queryKey: ['courses', formData.department_id],
    queryFn: () => fetchCourses(getDepartmentName(formData.department_id)),
    enabled: !!formData.department_id,
    staleTime: 10 * 60 * 1000,
  });

  // TanStack Query - Fetch instructors from staff API
  const {
    data: instructors = [],
    isLoading: isLoadingInstructors,
  } = useQuery({
    queryKey: ['instructors', formData.department_id],
    queryFn: () => fetchInstructors(getDepartmentName(formData.department_id)),
    enabled: !!formData.department_id,
    staleTime: 10 * 60 * 1000,
  });

  // TanStack Query - Fetch existing semester courses for the department
  const {
    data: existingSemesterCourses = [],
    isLoading: isLoadingExistingCourses,
  } = useQuery({
    queryKey: ['semesterCourses', semester, formData.department_id],
    queryFn: () => fetchSemesterCourses(semester, getDepartmentName(formData.department_id)),
    enabled: !!formData.department_id,
    staleTime: 5 * 60 * 1000,
  });

  // Day mapping for translating backend English days to Turkish
  const dayToTurkish: Record<string, string> = {
    monday: 'Pazartesi', tuesday: 'Salı', wednesday: 'Çarşamba',
    thursday: 'Perşembe', friday: 'Cuma',
  };

  // Mutation for creating semester course
  const createMutation = useMutation({
    mutationFn: (data: CreateSemesterCourseRequest) => createSemesterCourse(semester, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['semesterCourses', semester] });
      setSuccessDialog(true);
      setFormData(initialFormData);
      setSelectedCourse(null);
    },
    onError: async (error: any) => {
      console.error('Ders açılırken hata:', error);
      let code = '';
      let message = error.message;
      let details: { course_code?: string; department?: string; day_of_week?: string; slot_number?: number } | null = null;
      try {
        const body = await error.response?.json();
        if (body?.error) message = body.error;
        if (body?.code) code = body.code;
        if (body?.details) details = body.details;
      } catch { /* ignore parse errors */ }

      if (code === 'INSTRUCTOR_SCHEDULE_CONFLICT' && details?.department) {
        // Cross-department conflict with structured details
        const dayTR = dayToTurkish[details.day_of_week ?? ''] || details.day_of_week;
        setErrorDialog({
          open: true,
          title: 'Program Çakışması',
          message: `Öğretim görevlisi ${dayTR} ${details.slot_number}. slotta ${details.department} bölümünde ${details.course_code} dersini veriyor`,
        });
      } else if (code === 'INSTRUCTOR_SCHEDULE_CONFLICT') {
        // Same department conflict
        setErrorDialog({
          open: true,
          title: 'Program Çakışması',
          message: 'Öğretim görevlisinin bu saatlerde başka bir dersi bulunuyor',
        });
      } else {
        setErrorDialog({ open: true, title: 'Hata', message });
      }
    },
  });

  const isLoadingDepartmentData = isLoadingCourses || isLoadingInstructors;

  // Calculate occupied time slots from existing semester courses
  const occupiedSlots = React.useMemo(() => {
    const slots = new Set<string>(); // Format: "day_slot" e.g., "monday_1"

    existingSemesterCourses.forEach((course) => {
      course.schedule_sessions.forEach((session) => {
        session.slot_numbers.forEach((slotNum) => {
          slots.add(`${session.day_of_week}_${slotNum}`);
        });
      });
    });

    return slots;
  }, [existingSemesterCourses]);

  // Check if a specific slot is occupied
  const isSlotOccupied = (day: string, slot: number): boolean => {
    return occupiedSlots.has(`${day}_${slot}`);
  };

  // Get course code for an occupied slot
  const getOccupiedSlotCourse = (day: string, slot: number): string | null => {
    for (const course of existingSemesterCourses) {
      for (const session of course.schedule_sessions) {
        if (session.day_of_week === day && session.slot_numbers.includes(slot)) {
          return course.course_code;
        }
      }
    }
    return null;
  };

  // Update departments when faculty changes
  useEffect(() => {
    if (formData.faculty_id) {
      const faculty = mockFaculties.find(f => f.id === formData.faculty_id);
      if (faculty) {
        setDepartments(faculty.departments);
        setFormData(prev => ({ ...prev, department_id: '', course_id: '', instructor_id: '' }));
        setSelectedCourse(null);
      }
    } else {
      setDepartments([]);
      setSelectedCourse(null);
    }
  }, [formData.faculty_id]);

  // Reset form selections when department changes
  useEffect(() => {
    if (formData.department_id) {
      setFormData(prev => ({ ...prev, course_id: '', instructor_id: '' }));
      setSelectedCourse(null);
    }
  }, [formData.department_id]);

  // Update selected course info
  useEffect(() => {
    if (formData.course_id) {
      const course = courses.find(c => c.id === formData.course_id);
      if (course) {
        setSelectedCourse(course);
        setFormData(prev => ({ ...prev, class_level: course.class_level, schedule_sessions: [] }));
        // Auto-select available session type
        const hasLab = (course.lab_hours ?? 0) > 0;
        const hasTheory = (course.theoretical_hours ?? 0) > 0;
        if (hasTheory) setActiveSessionType('theory');
        else if (hasLab) setActiveSessionType('lab');
      }
    } else {
      setSelectedCourse(null);
    }
  }, [formData.course_id, courses]);

  // Update instructor fullname when instructor is selected
  useEffect(() => {
    if (formData.instructor_id) {
      const instructor = instructors.find((i: Instructor) => i.id === formData.instructor_id);
      if (instructor) {
        setFormData(prev => ({ ...prev, instructor_fullname: instructor.fullname }));
      }
    }
  }, [formData.instructor_id, instructors]);

  const handleInputChange = (field: keyof FormData, value: string | number) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  // Assessment schema management
  const addAssessment = () => {
    setFormData(prev => ({
      ...prev,
      assessment_schema: [...prev.assessment_schema, { slug: '', name: '', weight: 0 }],
    }));
  };

  const updateAssessment = (index: number, field: keyof AssessmentItem, value: string | number) => {
    setFormData(prev => ({
      ...prev,
      assessment_schema: prev.assessment_schema.map((item, i) => {
        if (i === index) {
          if (field === 'slug') {
            const predefined = predefinedAssessments.find(p => p.slug === value);
            return { ...item, slug: value as string, name: predefined?.name || item.name };
          }
          return { ...item, [field]: value };
        }
        return item;
      }),
    }));
  };

  const removeAssessment = (index: number) => {
    if (formData.assessment_schema.length > 1) {
      setFormData(prev => ({
        ...prev,
        assessment_schema: prev.assessment_schema.filter((_, i) => i !== index),
      }));
    }
  };

  // Schedule management - sessions are grouped by day + session_type
  const toggleScheduleSlot = (day: string, slot: number) => {
    setFormData(prev => {
      const key = (s: ScheduleSessionDTO) => s.day_of_week === day && s.session_type === activeSessionType;
      const existingSession = prev.schedule_sessions.find(key);

      // Check if we're adding (not removing) and if limit is reached
      const isRemoving = existingSession?.slot_numbers.includes(slot);
      if (!isRemoving) {
        const maxSlots = activeSessionType === 'theory' ? totalTheoryHours : totalLabHours;
        const currentUsed = prev.schedule_sessions
          .filter(s => s.session_type === activeSessionType)
          .reduce((sum, s) => sum + s.slot_numbers.length, 0);
        if (currentUsed >= maxSlots) return prev; // limit reached, don't add
      }

      if (existingSession) {
        const hasSlot = existingSession.slot_numbers.includes(slot);
        if (hasSlot) {
          // Remove slot
          const newSlots = existingSession.slot_numbers.filter(s => s !== slot);
          if (newSlots.length === 0) {
            return {
              ...prev,
              schedule_sessions: prev.schedule_sessions.filter(s => !key(s)),
            };
          }
          return {
            ...prev,
            schedule_sessions: prev.schedule_sessions.map(s =>
              key(s) ? { ...s, slot_numbers: newSlots.sort((a, b) => a - b) } : s
            ),
          };
        } else {
          // Add slot
          return {
            ...prev,
            schedule_sessions: prev.schedule_sessions.map(s =>
              key(s)
                ? { ...s, slot_numbers: [...s.slot_numbers, slot].sort((a, b) => a - b) }
                : s
            ),
          };
        }
      } else {
        // Also check if this slot is selected under a different session_type for the same day
        const otherSession = prev.schedule_sessions.find(
          s => s.day_of_week === day && s.session_type !== activeSessionType && s.slot_numbers.includes(slot)
        );
        if (otherSession) {
          // Remove from other session_type first
          const newOtherSlots = otherSession.slot_numbers.filter(s => s !== slot);
          const filtered = newOtherSlots.length === 0
            ? prev.schedule_sessions.filter(s => !(s.day_of_week === day && s.session_type !== activeSessionType))
            : prev.schedule_sessions.map(s =>
                s.day_of_week === day && s.session_type !== activeSessionType
                  ? { ...s, slot_numbers: newOtherSlots.sort((a, b) => a - b) }
                  : s
              );
          return {
            ...prev,
            schedule_sessions: [...filtered, { day_of_week: day, slot_numbers: [slot], session_type: activeSessionType }],
          };
        }
        // Create new session
        return {
          ...prev,
          schedule_sessions: [...prev.schedule_sessions, { day_of_week: day, slot_numbers: [slot], session_type: activeSessionType }],
        };
      }
    });
  };

  const isSlotSelected = (day: string, slot: number) => {
    return formData.schedule_sessions.some(s => s.day_of_week === day && s.slot_numbers.includes(slot));
  };

  const getSlotSessionType = (day: string, slot: number): 'theory' | 'lab' | null => {
    const session = formData.schedule_sessions.find(s => s.day_of_week === day && s.slot_numbers.includes(slot));
    return session?.session_type ?? null;
  };

  // Calculate used and remaining hours per session type
  const totalTheoryHours = selectedCourse?.theoretical_hours ?? 0;
  const totalLabHours = selectedCourse?.lab_hours ?? 0;
  const usedTheorySlots = formData.schedule_sessions
    .filter(s => s.session_type === 'theory')
    .reduce((sum, s) => sum + s.slot_numbers.length, 0);
  const usedLabSlots = formData.schedule_sessions
    .filter(s => s.session_type === 'lab')
    .reduce((sum, s) => sum + s.slot_numbers.length, 0);
  const remainingTheory = totalTheoryHours - usedTheorySlots;
  const remainingLab = totalLabHours - usedLabSlots;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!isWeightValid) {
      alert('Değerlendirme ağırlıkları toplamı %100 olmalıdır!');
      return;
    }

    if (formData.schedule_sessions.length === 0) {
      alert('En az bir ders saati seçmelisiniz!');
      return;
    }

    if (!selectedCourse) {
      alert('Lütfen bir ders seçin!');
      return;
    }

    // Prepare data for API - DTO uyumlu
    const requestData: CreateSemesterCourseRequest = {
      course_code: selectedCourse.course_code,
      class_level: selectedCourse.class_level,
      instructor_id: formData.instructor_id,
      instructor_fullname: formData.instructor_fullname,
      classroom_location: formData.classroom_location,
      max_capacity: formData.max_capacity,
      assessment_schema: formData.assessment_schema,
      schedule_sessions: formData.schedule_sessions,
    };

    console.log('Semester Course Data:', requestData);

    // Send to API
    createMutation.mutate(requestData);
  };

  return (
    <div className="container mx-auto py-6 px-4 max-w-5xl">
      {/* Header */}
      <div className="flex items-center gap-4 mb-6">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-600 text-white">
          <CalendarPlus className="h-5 w-5" />
        </div>
        <div>
          <h1 className="text-2xl font-bold dark:text-white">Dönem Ders Açılışı</h1>
          <p className="text-muted-foreground text-sm">Dönemlik açılacak dersleri tanımlayın</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Bölüm Seçimi */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GraduationCap className="h-5 w-5" />
              Bölüm Seçimi
            </CardTitle>
            <CardDescription>Dönemlik ders açılacak bölümü seçin</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Fakülte *</Label>
                <Select
                  value={formData.faculty_id}
                  onValueChange={(value) => handleInputChange('faculty_id', value)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Fakülte seçin" />
                  </SelectTrigger>
                  <SelectContent>
                    {mockFaculties.map((faculty) => (
                      <SelectItem key={faculty.id} value={faculty.id}>
                        {faculty.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label className="flex items-center gap-2">
                  Bölüm *
                  {isLoadingDepartmentData && <Loader2 className="h-3 w-3 animate-spin" />}
                </Label>
                <Select
                  value={formData.department_id}
                  onValueChange={(value) => handleInputChange('department_id', value)}
                  disabled={!formData.faculty_id}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={formData.faculty_id ? "Bölüm seçin" : "Önce fakülte seçin"} />
                  </SelectTrigger>
                  <SelectContent>
                    {departments.map((dept) => (
                      <SelectItem key={dept.id} value={dept.id}>
                        {dept.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Yeni Dönem Açılacak Ders */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BookOpen className="h-5 w-5" />
              Yeni Dönem Açılacak Ders
            </CardTitle>
            <CardDescription>Açılacak ders, öğretim üyesi ve sınıf bilgilerini girin</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Ders Seçimi */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label className="flex items-center gap-2">
                  <BookOpen className="h-4 w-4" />
                  Ders *
                  {isLoadingDepartmentData && <Loader2 className="h-3 w-3 animate-spin" />}
                </Label>
                <Select
                  value={formData.course_id}
                  onValueChange={(value) => handleInputChange('course_id', value)}
                  disabled={!formData.department_id || isLoadingDepartmentData}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={
                      isLoadingDepartmentData 
                        ? "Yükleniyor..." 
                        : formData.department_id 
                          ? "Ders seçin" 
                          : "Önce bölüm seçin"
                    } />
                  </SelectTrigger>
                  <SelectContent>
                    {courses.map((course) => (
                      <SelectItem key={course.id} value={course.id}>
                        {course.course_code} - {course.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label className="flex items-center gap-2">
                  <User className="h-4 w-4" />
                  Öğretim Üyesi *
                </Label>
                <Select
                  value={formData.instructor_id}
                  onValueChange={(value) => handleInputChange('instructor_id', value)}
                  disabled={!formData.department_id || isLoadingDepartmentData}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={
                      isLoadingDepartmentData 
                        ? "Yükleniyor..." 
                        : formData.department_id 
                          ? "Öğretim üyesi seçin" 
                          : "Önce bölüm seçin"
                    } />
                  </SelectTrigger>
                  <SelectContent>
                    {instructors.map((instructor: Instructor) => (
                      <SelectItem key={instructor.id} value={instructor.id}>
                        {instructor.fullname}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Selected Course Info */}
            {selectedCourse && (
              <div className="p-4 bg-muted/50 rounded-lg">
                <h4 className="font-medium mb-2">Seçilen Ders Bilgileri</h4>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <span className="text-muted-foreground">Ders Kodu:</span>
                    <p className="font-medium">{selectedCourse.course_code}</p>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Kredi:</span>
                    <p className="font-medium">{selectedCourse.credits}</p>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Sınıf:</span>
                    <p className="font-medium">{selectedCourse.class_level}. Sınıf</p>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Tür:</span>
                    <Badge variant={selectedCourse.course_type === 'mandatory' ? 'default' : 'secondary'}>
                      {selectedCourse.course_type === 'mandatory' ? 'Zorunlu' : 'Seçmeli'}
                    </Badge>
                  </div>
                </div>
              </div>
            )}

            <Separator />

            {/* Kontenjan ve Konum */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label className="flex items-center gap-2">
                  <Users className="h-4 w-4" />
                  Kontenjan *
                </Label>
                <Input
                  type="number"
                  min="1"
                  max="500"
                  placeholder="50"
                  value={formData.max_capacity}
                  onChange={(e) => handleInputChange('max_capacity', parseInt(e.target.value) || 50)}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label className="flex items-center gap-2">
                  <MapPin className="h-4 w-4" />
                  Derslik / Konum *
                </Label>
                <Input
                  placeholder="Örn: A Blok, Kat 2, Derslik 201"
                  value={formData.classroom_location}
                  onChange={(e) => handleInputChange('classroom_location', e.target.value)}
                  required
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Ders Programı */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Clock className="h-5 w-5" />
              Ders Programı
            </CardTitle>
            <CardDescription>Dersin hangi gün ve saatlerde yapılacağını seçin. Tip seçip slotlara tıklayın.</CardDescription>
          </CardHeader>
          <CardContent>
            {selectedCourse ? (
              <div className="flex items-center gap-3 mb-4">
                <Label>Ders Tipi:</Label>
                <div className="flex gap-2">
                  {totalTheoryHours > 0 && (
                    <Button
                      type="button"
                      size="sm"
                      variant={activeSessionType === 'theory' ? 'default' : 'outline'}
                      onClick={() => setActiveSessionType('theory')}
                    >
                      Teori ({remainingTheory}/{totalTheoryHours})
                    </Button>
                  )}
                  {totalLabHours > 0 && (
                    <Button
                      type="button"
                      size="sm"
                      variant={activeSessionType === 'lab' ? 'default' : 'outline'}
                      className={activeSessionType === 'lab' ? 'bg-emerald-600 hover:bg-emerald-700' : ''}
                      onClick={() => setActiveSessionType('lab')}
                    >
                      Lab ({remainingLab}/{totalLabHours})
                    </Button>
                  )}
                </div>
                {(totalTheoryHours > 0 || totalLabHours > 0) && (
                  <div className="flex items-center gap-3 ml-4 text-xs text-muted-foreground">
                    {totalTheoryHours > 0 && <span className="flex items-center gap-1"><span className="inline-block w-3 h-3 rounded bg-indigo-200 dark:bg-indigo-800" /> Teori</span>}
                    {totalLabHours > 0 && <span className="flex items-center gap-1"><span className="inline-block w-3 h-3 rounded bg-emerald-200 dark:bg-emerald-800" /> Lab</span>}
                  </div>
                )}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground mb-4">Ders seçildikten sonra program oluşturabilirsiniz.</p>
            )}
            {existingSemesterCourses.length > 0 && (
              <div className="mb-4 p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                <p className="text-sm text-blue-900 dark:text-blue-100 flex items-center gap-2">
                  <AlertCircle className="h-4 w-4" />
                  <span>
                    <strong>{existingSemesterCourses.length}</strong> ders bu dönem için açılmış.
                    Kırmızı alanlar dolu saatleri gösterir.
                  </span>
                </p>
              </div>
            )}
            <div className="overflow-x-auto">
              <table className="w-full border-collapse">
                <thead>
                  <tr>
                    <th className="border dark:border-gray-700 p-2 bg-muted text-left text-sm font-medium">Saat</th>
                    {daysOfWeek.map((day) => (
                      <th key={day.key} className="border dark:border-gray-700 p-2 bg-muted text-center text-sm font-medium min-w-[100px]">
                        {day.label}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {timeSlots.map((slot) => (
                    <tr key={slot.slot}>
                      <td className="border dark:border-gray-700 p-2 text-xs text-muted-foreground whitespace-nowrap">
                        {slot.slot}. Ders<br />
                        <span className="text-[10px]">{slot.time}</span>
                      </td>
                      {daysOfWeek.map((day) => {
                        const occupied = isSlotOccupied(day.key, slot.slot);
                        const selected = isSlotSelected(day.key, slot.slot);
                        const slotType = selected ? getSlotSessionType(day.key, slot.slot) : null;
                        const courseCode = occupied ? getOccupiedSlotCourse(day.key, slot.slot) : null;

                        return (
                          <td
                            key={day.key}
                            className={`border dark:border-gray-700 p-1 text-center ${
                              occupied ? 'bg-red-50 dark:bg-red-950/20'
                              : slotType === 'theory' ? 'bg-indigo-50 dark:bg-indigo-950/20'
                              : slotType === 'lab' ? 'bg-emerald-50 dark:bg-emerald-950/20'
                              : ''
                            }`}
                            title={occupied ? `Dolu: ${courseCode}` : slotType ? `${slotType === 'theory' ? 'Teori' : 'Lab'}` : ''}
                          >
                            {occupied ? (
                              <div className="flex items-center justify-center h-full py-2">
                                <span className="text-[10px] font-semibold text-red-600 dark:text-red-400 leading-tight text-center">
                                  {courseCode}
                                </span>
                              </div>
                            ) : (
                              <Checkbox
                                checked={selected}
                                onCheckedChange={() => toggleScheduleSlot(day.key, slot.slot)}
                                className="h-5 w-5"
                              />
                            )}
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Selected schedule summary */}
            {formData.schedule_sessions.length > 0 && (
              <div className="mt-4 p-3 bg-muted/50 rounded-lg">
                <h4 className="text-sm font-medium mb-2">Seçilen Saatler:</h4>
                <div className="flex flex-wrap gap-2">
                  {formData.schedule_sessions.map((session) => {
                    const dayLabel = daysOfWeek.find(d => d.key === session.day_of_week)?.label;
                    const typeLabel = session.session_type === 'lab' ? 'Lab' : 'Teori';
                    return session.slot_numbers.map((slot) => (
                      <Badge
                        key={`${session.day_of_week}-${session.session_type}-${slot}`}
                        variant="secondary"
                        className={session.session_type === 'lab' ? 'bg-emerald-100 dark:bg-emerald-900 text-emerald-800 dark:text-emerald-200' : 'bg-indigo-100 dark:bg-indigo-900 text-indigo-800 dark:text-indigo-200'}
                      >
                        {dayLabel} {slot}. Ders ({typeLabel})
                      </Badge>
                    ));
                  })}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Değerlendirme Şeması */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Percent className="h-5 w-5" />
              Değerlendirme Şeması
            </CardTitle>
            <CardDescription>
              Not değerlendirme ağırlıklarını belirleyin (Toplam %100 olmalıdır)
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {formData.assessment_schema.map((assessment, index) => (
              <div key={index} className="flex gap-3 items-end">
                <div className="flex-1 space-y-2">
                  <Label>Değerlendirme Türü</Label>
                  <Select
                    value={assessment.slug}
                    onValueChange={(value) => updateAssessment(index, 'slug', value)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Tür seçin" />
                    </SelectTrigger>
                    <SelectContent>
                      {predefinedAssessments.map((type) => (
                        <SelectItem key={type.slug} value={type.slug}>
                          {type.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="w-32 space-y-2">
                  <Label>Ağırlık (%)</Label>
                  <Input
                    type="number"
                    min="0"
                    max="100"
                    value={assessment.weight}
                    onChange={(e) => updateAssessment(index, 'weight', parseInt(e.target.value) || 0)}
                  />
                </div>

                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => removeAssessment(index)}
                  disabled={formData.assessment_schema.length === 1}
                  className="text-destructive hover:text-destructive"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}

            <div className="flex items-center justify-between pt-2">
              <Button type="button" variant="outline" size="sm" onClick={addAssessment}>
                <Plus className="h-4 w-4 mr-2" />
                Değerlendirme Ekle
              </Button>

              <div className={`flex items-center gap-2 text-sm font-medium ${isWeightValid ? 'text-green-600' : 'text-destructive'}`}>
                {isWeightValid ? (
                  <CheckCircle2 className="h-4 w-4" />
                ) : (
                  <AlertCircle className="h-4 w-4" />
                )}
                Toplam: %{totalWeight}
                {!isWeightValid && <span className="text-xs">(%100 olmalı)</span>}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Submit Buttons */}
        <div className="flex justify-end gap-4 pb-8">
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setFormData(initialFormData);
              setSelectedCourse(null);
            }}
          >
            Temizle
          </Button>
          <Button
            type="submit"
            disabled={createMutation.isPending || !selectedCourse || !formData.instructor_id || !isWeightValid}
          >
            <Save className="h-4 w-4 mr-2" />
            {createMutation.isPending ? 'Kaydediliyor...' : 'Dersi Aç'}
          </Button>
        </div>
      </form>

      {/* Error Dialog */}
      <AlertDialog open={errorDialog.open} onOpenChange={(open) => setErrorDialog(prev => ({ ...prev, open }))}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2 text-red-600">
              <AlertCircle className="h-5 w-5" />
              {errorDialog.title}
            </AlertDialogTitle>
            <AlertDialogDescription className="text-base">
              {errorDialog.message}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogAction>Anladım</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Success Dialog */}
      <AlertDialog open={successDialog} onOpenChange={setSuccessDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2 text-green-600">
              <CheckCircle2 className="h-5 w-5" />
              Başarılı
            </AlertDialogTitle>
            <AlertDialogDescription className="text-base">
              Dönemlik ders başarıyla açıldı!
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogAction>Anladım</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
