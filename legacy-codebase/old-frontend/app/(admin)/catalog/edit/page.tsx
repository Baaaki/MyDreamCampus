'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { mockFaculties } from '@/mock_data/catalog';
import type { CourseCatalog, Department, Faculty, WeeklyTopic, CourseCoordinator } from '@/lib/types';
import { catalogService, type UpdateCourseRequest } from '@/lib/services/catalog-service';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
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
import {
  Book,
  Clock,
  GraduationCap,
  Calendar,
  ChevronDown,
  ChevronRight,
  ArrowLeft,
  BookOpen,
  Building2,
  Edit3,
  Plus,
  Trash2,
  User,
  Library,
  Save,
  Loader2,
  AlertCircle,
  X,
} from 'lucide-react';

// Form data interface
interface FormData {
  course_code: string;
  name: string;
  faculty: string;
  department: string;
  offering_unit: string;
  semester: number;
  class_level: number;
  credits: number;
  theoretical_hours: number;
  lab_hours: number;
  ects: number;
  course_type: string;
  course_category: string;
  education_level: string;
  teaching_type: string;
  language: string;
  coordinator: CourseCoordinator;
  purpose: string;
  learning_outcomes_list: string[];
  weekly_topics: WeeklyTopic[];
  recommended_sources: string[];
  description: string;
}

// Helper to get semester info
const getSemesterInfo = (semester: number) => {
  const year = Math.ceil(semester / 2);
  const term = semester % 2 === 1 ? 'Güz' : 'Bahar';
  return { year, term, label: `${year}. Yıl - ${term} Dönemi` };
};

// Convert course to form data
const courseToFormData = (course: CourseCatalog): FormData => ({
  course_code: course.course_code,
  name: course.name,
  faculty: course.faculty,
  department: course.department,
  offering_unit: course.offering_unit || '',
  semester: course.semester || 1,
  class_level: course.class_level,
  credits: course.credits,
  theoretical_hours: course.theoretical_hours,
  lab_hours: course.lab_hours || 0,
  ects: course.ects || course.credits,
  course_type: course.course_type,
  course_category: 'theoretical',
  education_level: course.education_level || 'Lisans',
  teaching_type: course.teaching_type || 'Örgün Öğretim',
  language: course.language || 'Türkçe',
  coordinator: course.coordinator || { title: '', name: '', email: '', phone: '', office: '' },
  purpose: course.purpose || '',
  learning_outcomes_list: course.learning_outcomes_list?.length ? course.learning_outcomes_list : [''],
  weekly_topics: course.weekly_topics?.length ? course.weekly_topics : [{ week: 1, topic: '', description: '' }],
  recommended_sources: course.recommended_sources?.length ? course.recommended_sources : [''],
  description: course.description || '',
});

// Map Turkish display values to backend enum values
const mapEducationLevelToBackend = (level: string): string => {
  const mapping: Record<string, string> = {
    'Lisans': 'undergraduate',
    'Yüksek Lisans': 'graduate',
    'Doktora': 'doctorate',
  };
  return mapping[level] || 'undergraduate';
};

const mapTeachingTypeToBackend = (type: string): string => {
  const mapping: Record<string, string> = {
    'Örgün Öğretim': 'on_campus',
    'Uzaktan Öğretim': 'online',
    'Hibrit': 'hybrid',
  };
  return mapping[type] || 'on_campus';
};

export default function EditCourseCatalogPage() {
  const router = useRouter();
  const [expandedFaculties, setExpandedFaculties] = useState<string[]>([]);
  const [selectedDepartment, setSelectedDepartment] = useState<{ dept: Department; faculty: Faculty } | null>(null);

  // API state
  const [departmentCourses, setDepartmentCourses] = useState<CourseCatalog[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [courseCounts, setCourseCounts] = useState<Record<string, number>>({});

  // Edit state
  const [editingCourse, setEditingCourse] = useState<CourseCatalog | null>(null);
  const [formData, setFormData] = useState<FormData | null>(null);
  const [originalFormData, setOriginalFormData] = useState<FormData | null>(null);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);

  // Toggle faculty accordion
  const toggleFaculty = (facultyId: string) => {
    setExpandedFaculties(prev =>
      prev.includes(facultyId)
        ? prev.filter(id => id !== facultyId)
        : [...prev, facultyId]
    );
  };

  // Fetch courses when department is selected
  const fetchDepartmentCourses = useCallback(async (departmentName: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const courses = await catalogService.getCoursesByDepartment(departmentName);
      setDepartmentCourses(courses);
    } catch (err) {
      console.error('Failed to fetch courses:', err);
      setError('Dersler yüklenirken bir hata oluştu.');
      setDepartmentCourses([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch course counts
  const fetchCourseCounts = useCallback(async () => {
    try {
      const { courses } = await catalogService.listCourses({ limit: 100 });
      const counts: Record<string, number> = {};
      courses.forEach(course => {
        counts[course.department] = (counts[course.department] || 0) + 1;
      });
      setCourseCounts(counts);
    } catch (err) {
      console.error('Failed to fetch course counts:', err);
    }
  }, []);

  useEffect(() => {
    fetchCourseCounts();
  }, [fetchCourseCounts]);

  useEffect(() => {
    if (selectedDepartment) {
      fetchDepartmentCourses(selectedDepartment.dept.name);
    }
  }, [selectedDepartment, fetchDepartmentCourses]);

  // Group courses by semester
  const groupCoursesBySemester = (courses: CourseCatalog[]) => {
    const grouped: { [key: number]: { mandatory: CourseCatalog[], elective: CourseCatalog[] } } = {};
    for (let i = 1; i <= 8; i++) {
      grouped[i] = { mandatory: [], elective: [] };
    }
    courses.forEach(course => {
      const semester = course.semester || 1;
      if (course.course_type === 'mandatory') {
        grouped[semester].mandatory.push(course);
      } else {
        grouped[semester].elective.push(course);
      }
    });
    return grouped;
  };

  // Open edit dialog
  const handleEditClick = async (course: CourseCatalog) => {
    try {
      // Fetch full course details
      const fullCourse = await catalogService.getCourseByCode(course.course_code);
      setEditingCourse(fullCourse);
      const formDataFromCourse = courseToFormData(fullCourse);
      setFormData(formDataFromCourse);
      setOriginalFormData(JSON.parse(JSON.stringify(formDataFromCourse)));
      setIsEditDialogOpen(true);
    } catch (err) {
      console.error('Failed to fetch course details:', err);
      alert('Ders bilgileri yüklenemedi!');
    }
  };

  // Check if form has changes
  const hasChanges = useMemo(() => {
    if (!formData || !originalFormData) return false;
    return JSON.stringify(formData) !== JSON.stringify(originalFormData);
  }, [formData, originalFormData]);

  // Form handlers
  const handleInputChange = (field: keyof FormData, value: string | number) => {
    if (!formData) return;
    setFormData(prev => prev ? { ...prev, [field]: value } : null);
  };

  const handleCoordinatorChange = (field: keyof CourseCoordinator, value: string) => {
    if (!formData) return;
    setFormData(prev => prev ? {
      ...prev,
      coordinator: { ...prev.coordinator, [field]: value },
    } : null);
  };

  // Learning outcomes
  const addLearningOutcome = () => {
    if (!formData) return;
    setFormData(prev => prev ? {
      ...prev,
      learning_outcomes_list: [...prev.learning_outcomes_list, ''],
    } : null);
  };

  const updateLearningOutcome = (index: number, value: string) => {
    if (!formData) return;
    setFormData(prev => prev ? {
      ...prev,
      learning_outcomes_list: prev.learning_outcomes_list.map((item, i) => i === index ? value : item),
    } : null);
  };

  const removeLearningOutcome = (index: number) => {
    if (!formData || formData.learning_outcomes_list.length <= 1) return;
    setFormData(prev => prev ? {
      ...prev,
      learning_outcomes_list: prev.learning_outcomes_list.filter((_, i) => i !== index),
    } : null);
  };

  // Weekly topics
  const addWeeklyTopic = () => {
    if (!formData) return;
    const nextWeek = formData.weekly_topics.length + 1;
    setFormData(prev => prev ? {
      ...prev,
      weekly_topics: [...prev.weekly_topics, { week: nextWeek, topic: '', description: '' }],
    } : null);
  };

  const updateWeeklyTopic = (index: number, field: keyof WeeklyTopic, value: string | number) => {
    if (!formData) return;
    setFormData(prev => prev ? {
      ...prev,
      weekly_topics: prev.weekly_topics.map((item, i) => i === index ? { ...item, [field]: value } : item),
    } : null);
  };

  const removeWeeklyTopic = (index: number) => {
    if (!formData || formData.weekly_topics.length <= 1) return;
    setFormData(prev => prev ? {
      ...prev,
      weekly_topics: prev.weekly_topics.filter((_, i) => i !== index).map((item, i) => ({ ...item, week: i + 1 })),
    } : null);
  };

  // Sources
  const addSource = () => {
    if (!formData) return;
    setFormData(prev => prev ? {
      ...prev,
      recommended_sources: [...prev.recommended_sources, ''],
    } : null);
  };

  const updateSource = (index: number, value: string) => {
    if (!formData) return;
    setFormData(prev => prev ? {
      ...prev,
      recommended_sources: prev.recommended_sources.map((item, i) => i === index ? value : item),
    } : null);
  };

  const removeSource = (index: number) => {
    if (!formData || formData.recommended_sources.length <= 1) return;
    setFormData(prev => prev ? {
      ...prev,
      recommended_sources: prev.recommended_sources.filter((_, i) => i !== index),
    } : null);
  };

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: async ({ courseCode, data }: { courseCode: string; data: UpdateCourseRequest }) => {
      return catalogService.updateCourse(courseCode, data);
    },
    onSuccess: () => {
      alert('Ders başarıyla güncellendi!');
      setIsEditDialogOpen(false);
      setEditingCourse(null);
      setFormData(null);
      setOriginalFormData(null);
      // Refresh courses
      if (selectedDepartment) {
        fetchDepartmentCourses(selectedDepartment.dept.name);
      }
    },
    onError: (error: Error) => {
      console.error('Failed to update course:', error);
      alert(`Güncelleme hatası: ${error.message}`);
    },
  });

  // Submit handler
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData || !editingCourse) return;

    const updateData: UpdateCourseRequest = {
      name: formData.name,
      faculty: formData.faculty,
      department: formData.department,
      offering_unit: formData.offering_unit || undefined,
      class_level: formData.class_level,
      semester: formData.semester,
      credits: formData.credits,
      ects: formData.ects,
      theoretical_hours: formData.theoretical_hours,
      lab_hours: formData.lab_hours,
      course_type: formData.course_type as 'mandatory' | 'elective',
      education_level: mapEducationLevelToBackend(formData.education_level),
      teaching_type: mapTeachingTypeToBackend(formData.teaching_type),
      language: formData.language,
      coordinator: formData.coordinator.name ? formData.coordinator : undefined,
      purpose: formData.purpose || undefined,
      learning_outcomes_list: formData.learning_outcomes_list.filter(o => o.trim() !== ''),
      weekly_topics: formData.weekly_topics.filter(t => t.topic.trim() !== ''),
      recommended_sources: formData.recommended_sources.filter(s => s.trim() !== ''),
    };

    updateMutation.mutate({ courseCode: editingCourse.course_code, data: updateData });
  };

  const handleDepartmentClick = (dept: Department, faculty: Faculty) => {
    setSelectedDepartment({ dept, faculty });
  };

  const closeEditDialog = () => {
    if (hasChanges) {
      if (!confirm('Kaydedilmemiş değişiklikler var. Çıkmak istediğinizden emin misiniz?')) {
        return;
      }
    }
    setIsEditDialogOpen(false);
    setEditingCourse(null);
    setFormData(null);
    setOriginalFormData(null);
  };

  // Faculty & Department List View
  if (!selectedDepartment) {
    return (
      <div className="min-h-screen bg-gray-50 py-8">
        <div className="max-w-4xl mx-auto px-4">
          <div className="bg-white rounded-lg shadow-md p-6">
            <div className="flex items-center gap-4 mb-8">
              <Button variant="ghost" size="sm" onClick={() => router.push('/catalog')}>
                <ArrowLeft className="h-4 w-4 mr-2" />
                Geri
              </Button>
              <div className="flex-1">
                <h1 className="text-2xl font-bold text-gray-900">Ders Düzenle</h1>
                <p className="text-gray-600 text-sm">Fakülte ve bölüm seçerek dersleri düzenleyebilirsiniz</p>
              </div>
              <Badge variant="outline" className="flex items-center gap-1">
                <Edit3 className="h-3 w-3" />
                Düzenleme Modu
              </Badge>
            </div>

            {/* Faculty Accordion */}
            <div className="space-y-2">
              {mockFaculties.map((faculty) => {
                const isExpanded = expandedFaculties.includes(faculty.id);
                return (
                  <div key={faculty.id} className="border rounded-lg overflow-hidden">
                    <button
                      onClick={() => toggleFaculty(faculty.id)}
                      className="w-full flex items-center justify-between p-4 bg-gray-50 hover:bg-gray-100 transition-colors text-left"
                    >
                      <div className="flex items-center gap-3">
                        <Building2 className="h-5 w-5 text-indigo-600" />
                        <span className="font-semibold text-gray-900">{faculty.name}</span>
                        <Badge variant="outline" className="text-xs">
                          {faculty.departments.length} bölüm
                        </Badge>
                      </div>
                      {isExpanded ? (
                        <ChevronDown className="h-5 w-5 text-gray-500" />
                      ) : (
                        <ChevronRight className="h-5 w-5 text-gray-500" />
                      )}
                    </button>

                    {isExpanded && (
                      <div className="border-t bg-white">
                        {faculty.departments.map((dept, index) => {
                          const courseCount = courseCounts[dept.name] || 0;
                          return (
                            <button
                              key={dept.id}
                              onClick={() => handleDepartmentClick(dept, faculty)}
                              className={`w-full flex items-center justify-between p-3 pl-12 hover:bg-indigo-50 transition-colors text-left ${
                                index !== faculty.departments.length - 1 ? 'border-b border-gray-100' : ''
                              }`}
                            >
                              <div className="flex items-center gap-3">
                                <GraduationCap className="h-4 w-4 text-gray-400" />
                                <span className="text-gray-700">{dept.name}</span>
                              </div>
                              <div className="flex items-center gap-2">
                                {courseCount > 0 && (
                                  <span className="text-xs text-gray-500">{courseCount} ders</span>
                                )}
                                <ChevronRight className="h-4 w-4 text-gray-400" />
                              </div>
                            </button>
                          );
                        })}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Department Detail View
  const groupedCourses = groupCoursesBySemester(departmentCourses);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 py-8">
        <div className="max-w-7xl mx-auto px-4">
          <div className="bg-white rounded-lg shadow-md p-6">
            <button
              onClick={() => setSelectedDepartment(null)}
              className="flex items-center gap-2 text-indigo-600 hover:text-indigo-800 mb-4"
            >
              <ArrowLeft className="h-5 w-5" />
              <span>Tüm Fakülteler</span>
            </button>
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-12 w-12 animate-spin text-indigo-600" />
              <span className="ml-4 text-gray-600">Dersler yükleniyor...</span>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 py-8">
        <div className="max-w-7xl mx-auto px-4">
          <div className="bg-white rounded-lg shadow-md p-6">
            <button
              onClick={() => setSelectedDepartment(null)}
              className="flex items-center gap-2 text-indigo-600 hover:text-indigo-800 mb-4"
            >
              <ArrowLeft className="h-5 w-5" />
              <span>Tüm Fakülteler</span>
            </button>
            <div className="text-center py-12">
              <AlertCircle className="h-16 w-16 mx-auto text-red-300 mb-4" />
              <h2 className="text-xl font-semibold text-gray-700 mb-2">Hata Oluştu</h2>
              <p className="text-gray-500 mb-4">{error}</p>
              <Button onClick={() => fetchDepartmentCourses(selectedDepartment.dept.name)} variant="outline">
                Tekrar Dene
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-7xl mx-auto px-4">
        {/* Header */}
        <div className="bg-white rounded-lg shadow-md p-6 mb-6">
          <button
            onClick={() => setSelectedDepartment(null)}
            className="flex items-center gap-2 text-indigo-600 hover:text-indigo-800 mb-4 transition-colors"
          >
            <ArrowLeft className="h-5 w-5" />
            <span>Tüm Fakülteler</span>
          </button>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-16 h-16 bg-indigo-100 rounded-xl flex items-center justify-center">
                <Edit3 className="h-8 w-8 text-indigo-600" />
              </div>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">{selectedDepartment.dept.name}</h1>
                <p className="text-gray-600">{selectedDepartment.faculty.name}</p>
                <p className="text-sm text-gray-500 mt-1">{departmentCourses.length} ders - Düzenleme Modu</p>
              </div>
            </div>
            <Badge variant="outline" className="flex items-center gap-1 text-amber-600 border-amber-300 bg-amber-50">
              <Edit3 className="h-3 w-3" />
              Düzenleme
            </Badge>
          </div>
        </div>

        {/* Courses */}
        {departmentCourses.length === 0 ? (
          <div className="bg-white rounded-lg shadow-md p-8 text-center">
            <BookOpen className="h-16 w-16 text-gray-300 mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-gray-700 mb-2">Ders Bilgisi Bulunamadı</h2>
            <p className="text-gray-500">Bu bölüm için henüz ders bilgisi eklenmemiş.</p>
          </div>
        ) : (
          <div className="space-y-8">
            {[1, 2, 3, 4].map((year) => (
              <div key={year} className="bg-white rounded-lg shadow-md overflow-hidden">
                <div className="bg-gradient-to-r from-indigo-600 to-indigo-700 px-6 py-4">
                  <h2 className="text-xl font-bold text-white">{year}. Yıl</h2>
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-2 divide-y lg:divide-y-0 lg:divide-x divide-gray-200">
                  {[year * 2 - 1, year * 2].map((semester) => {
                    const semesterInfo = getSemesterInfo(semester);
                    const semesterCourses = groupedCourses[semester];
                    const totalEcts = [...semesterCourses.mandatory, ...semesterCourses.elective]
                      .reduce((sum, c) => sum + (c.ects || c.credits), 0);

                    return (
                      <div key={semester} className="p-6">
                        <div className="flex items-center justify-between mb-4">
                          <h3 className="text-lg font-semibold text-gray-800">
                            {semester}. Dönem ({semesterInfo.term})
                          </h3>
                          <Badge variant="outline" className="text-sm">
                            {totalEcts} AKTS
                          </Badge>
                        </div>

                        {/* Zorunlu Dersler */}
                        {semesterCourses.mandatory.length > 0 && (
                          <div className="mb-6">
                            <h4 className="text-sm font-semibold text-red-700 mb-3 flex items-center gap-2">
                              <div className="w-2 h-2 bg-red-500 rounded-full"></div>
                              ZORUNLU DERSLER
                            </h4>
                            <div className="overflow-x-auto">
                              <table className="w-full text-sm">
                                <thead>
                                  <tr className="bg-red-50 text-left">
                                    <th className="px-3 py-2 font-medium text-gray-700">Ders Kodu</th>
                                    <th className="px-3 py-2 font-medium text-gray-700">Ders Adı</th>
                                    <th className="px-3 py-2 font-medium text-gray-700 text-center">D</th>
                                    <th className="px-3 py-2 font-medium text-gray-700 text-center">AKTS</th>
                                    <th className="px-3 py-2 font-medium text-gray-700 text-center">İşlem</th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {semesterCourses.mandatory.map((course) => (
                                    <tr
                                      key={course.id}
                                      className="border-b border-gray-100 hover:bg-red-50 transition-colors"
                                    >
                                      <td className="px-3 py-2 font-medium text-indigo-600">{course.course_code}</td>
                                      <td className="px-3 py-2 text-gray-800">{course.name}</td>
                                      <td className="px-3 py-2 text-center text-gray-600">{course.theoretical_hours}</td>
                                      <td className="px-3 py-2 text-center font-medium text-gray-800">{course.ects || course.credits}</td>
                                      <td className="px-3 py-2 text-center">
                                        <Button
                                          size="sm"
                                          variant="ghost"
                                          onClick={() => handleEditClick(course)}
                                          className="h-8 w-8 p-0 hover:bg-indigo-100"
                                        >
                                          <Edit3 className="h-4 w-4 text-indigo-600" />
                                        </Button>
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          </div>
                        )}

                        {/* Seçmeli Dersler */}
                        {semesterCourses.elective.length > 0 && (
                          <div>
                            <h4 className="text-sm font-semibold text-green-700 mb-3 flex items-center gap-2">
                              <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                              SEÇMELİ DERSLER
                            </h4>
                            <div className="overflow-x-auto">
                              <table className="w-full text-sm">
                                <thead>
                                  <tr className="bg-green-50 text-left">
                                    <th className="px-3 py-2 font-medium text-gray-700">Ders Kodu</th>
                                    <th className="px-3 py-2 font-medium text-gray-700">Ders Adı</th>
                                    <th className="px-3 py-2 font-medium text-gray-700 text-center">D</th>
                                    <th className="px-3 py-2 font-medium text-gray-700 text-center">AKTS</th>
                                    <th className="px-3 py-2 font-medium text-gray-700 text-center">İşlem</th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {semesterCourses.elective.map((course) => (
                                    <tr
                                      key={course.id}
                                      className="border-b border-gray-100 hover:bg-green-50 transition-colors"
                                    >
                                      <td className="px-3 py-2 font-medium text-indigo-600">{course.course_code}</td>
                                      <td className="px-3 py-2 text-gray-800">{course.name}</td>
                                      <td className="px-3 py-2 text-center text-gray-600">{course.theoretical_hours}</td>
                                      <td className="px-3 py-2 text-center font-medium text-gray-800">{course.ects || course.credits}</td>
                                      <td className="px-3 py-2 text-center">
                                        <Button
                                          size="sm"
                                          variant="ghost"
                                          onClick={() => handleEditClick(course)}
                                          className="h-8 w-8 p-0 hover:bg-indigo-100"
                                        >
                                          <Edit3 className="h-4 w-4 text-indigo-600" />
                                        </Button>
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          </div>
                        )}

                        {semesterCourses.mandatory.length === 0 && semesterCourses.elective.length === 0 && (
                          <p className="text-sm text-gray-400 italic">Bu dönem için ders bilgisi yok</p>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Edit Dialog */}
      <Dialog open={isEditDialogOpen} onOpenChange={(open) => !open && closeEditDialog()}>
        <DialogContent className="sm:max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <div className="flex items-center justify-between">
              <div>
                <DialogTitle className="text-xl flex items-center gap-2">
                  <Edit3 className="h-5 w-5 text-indigo-600" />
                  {editingCourse?.course_code} - {editingCourse?.name}
                </DialogTitle>
                <DialogDescription>
                  Ders bilgilerini düzenleyin
                </DialogDescription>
              </div>
              {hasChanges && (
                <Badge variant="outline" className="text-amber-600 border-amber-300 bg-amber-50">
                  <AlertCircle className="h-3 w-3 mr-1" />
                  Değişiklikler var
                </Badge>
              )}
            </div>
          </DialogHeader>

          {formData && (
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Temel Bilgiler */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center gap-2">
                    <BookOpen className="h-4 w-4" />
                    Temel Bilgiler
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Ders Kodu</Label>
                      <Input value={formData.course_code} disabled className="bg-gray-100" />
                    </div>
                    <div className="space-y-2">
                      <Label>Ders Adı *</Label>
                      <Input
                        value={formData.name}
                        onChange={(e) => handleInputChange('name', e.target.value)}
                        required
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label>Dersi Veren Birim</Label>
                    <Input
                      value={formData.offering_unit}
                      onChange={(e) => handleInputChange('offering_unit', e.target.value)}
                      placeholder="Örn: Atatürk İlkeleri ve İnkılap Tarihi Bölümü"
                    />
                  </div>

                  <Separator />

                  <div className="grid grid-cols-4 gap-4">
                    <div className="space-y-2">
                      <Label>Dönem</Label>
                      <Select
                        value={formData.semester.toString()}
                        onValueChange={(v) => handleInputChange('semester', parseInt(v))}
                      >
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          {[1, 2, 3, 4, 5, 6, 7, 8].map((s) => (
                            <SelectItem key={s} value={s.toString()}>{s}. Dönem</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>Sınıf</Label>
                      <Select
                        value={formData.class_level.toString()}
                        onValueChange={(v) => handleInputChange('class_level', parseInt(v))}
                      >
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          {[1, 2, 3, 4].map((l) => (
                            <SelectItem key={l} value={l.toString()}>{l}. Sınıf</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>Ders Türü</Label>
                      <Select
                        value={formData.course_type}
                        onValueChange={(v) => handleInputChange('course_type', v)}
                      >
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="mandatory">Zorunlu</SelectItem>
                          <SelectItem value="elective">Seçmeli</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>Dil</Label>
                      <Select
                        value={formData.language}
                        onValueChange={(v) => handleInputChange('language', v)}
                      >
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Türkçe">Türkçe</SelectItem>
                          <SelectItem value="İngilizce">İngilizce</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <Separator />

                  <div className="grid grid-cols-4 gap-4">
                    <div className="space-y-2">
                      <Label>Teorik</Label>
                      <Input
                        type="number"
                        min="0"
                        value={formData.theoretical_hours}
                        onChange={(e) => handleInputChange('theoretical_hours', parseInt(e.target.value) || 0)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Lab</Label>
                      <Input
                        type="number"
                        min="0"
                        value={formData.lab_hours}
                        onChange={(e) => handleInputChange('lab_hours', parseInt(e.target.value) || 0)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Kredi</Label>
                      <Input
                        type="number"
                        min="0"
                        value={formData.credits}
                        onChange={(e) => handleInputChange('credits', parseInt(e.target.value) || 0)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>AKTS</Label>
                      <Input
                        type="number"
                        min="0"
                        value={formData.ects}
                        onChange={(e) => handleInputChange('ects', parseInt(e.target.value) || 0)}
                      />
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Koordinatör */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center gap-2">
                    <User className="h-4 w-4" />
                    Ders Koordinatörü
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Unvan</Label>
                      <Select
                        value={formData.coordinator.title}
                        onValueChange={(v) => handleCoordinatorChange('title', v)}
                      >
                        <SelectTrigger><SelectValue placeholder="Unvan seçin" /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Prof. Dr.">Prof. Dr.</SelectItem>
                          <SelectItem value="Doç. Dr.">Doç. Dr.</SelectItem>
                          <SelectItem value="Dr. Öğr. Üyesi">Dr. Öğr. Üyesi</SelectItem>
                          <SelectItem value="Öğr. Gör.">Öğr. Gör.</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>Ad Soyad</Label>
                      <Input
                        value={formData.coordinator.name}
                        onChange={(e) => handleCoordinatorChange('name', e.target.value)}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <div className="space-y-2">
                      <Label>E-posta</Label>
                      <Input
                        type="email"
                        value={formData.coordinator.email}
                        onChange={(e) => handleCoordinatorChange('email', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Telefon</Label>
                      <Input
                        value={formData.coordinator.phone}
                        onChange={(e) => handleCoordinatorChange('phone', e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>Ofis</Label>
                      <Input
                        value={formData.coordinator.office}
                        onChange={(e) => handleCoordinatorChange('office', e.target.value)}
                      />
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Ders İçeriği */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center gap-2">
                    <BookOpen className="h-4 w-4" />
                    Ders İçeriği
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label>Dersin Amacı</Label>
                    <Textarea
                      value={formData.purpose}
                      onChange={(e) => handleInputChange('purpose', e.target.value)}
                      rows={3}
                    />
                  </div>
                </CardContent>
              </Card>

              {/* Öğrenme Kazanımları */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center gap-2">
                    <GraduationCap className="h-4 w-4" />
                    Öğrenme Kazanımları
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {formData.learning_outcomes_list.map((outcome, index) => (
                    <div key={index} className="flex gap-2 items-center">
                      <span className="w-6 h-6 flex items-center justify-center bg-muted rounded text-xs font-medium">
                        {index + 1}
                      </span>
                      <Input
                        value={outcome}
                        onChange={(e) => updateLearningOutcome(index, e.target.value)}
                        className="flex-1"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => removeLearningOutcome(index)}
                        disabled={formData.learning_outcomes_list.length === 1}
                      >
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  ))}
                  <Button type="button" variant="outline" size="sm" onClick={addLearningOutcome}>
                    <Plus className="h-4 w-4 mr-1" />
                    Kazanım Ekle
                  </Button>
                </CardContent>
              </Card>

              {/* Haftalık Konular */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center gap-2">
                    <Calendar className="h-4 w-4" />
                    Haftalık Konular
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {formData.weekly_topics.map((topic, index) => (
                    <div key={index} className="p-3 border rounded-lg space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-sm">Hafta {topic.week}</span>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => removeWeeklyTopic(index)}
                          disabled={formData.weekly_topics.length === 1}
                        >
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      </div>
                      <Input
                        value={topic.topic}
                        onChange={(e) => updateWeeklyTopic(index, 'topic', e.target.value)}
                        placeholder="Konu başlığı"
                      />
                    </div>
                  ))}
                  <Button type="button" variant="outline" size="sm" onClick={addWeeklyTopic}>
                    <Plus className="h-4 w-4 mr-1" />
                    Hafta Ekle
                  </Button>
                </CardContent>
              </Card>

              {/* Kaynaklar */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center gap-2">
                    <Library className="h-4 w-4" />
                    Önerilen Kaynaklar
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {formData.recommended_sources.map((source, index) => (
                    <div key={index} className="flex gap-2 items-center">
                      <span className="w-6 h-6 flex items-center justify-center bg-muted rounded text-xs font-medium">
                        {index + 1}
                      </span>
                      <Input
                        value={source}
                        onChange={(e) => updateSource(index, e.target.value)}
                        className="flex-1"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => removeSource(index)}
                        disabled={formData.recommended_sources.length === 1}
                      >
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  ))}
                  <Button type="button" variant="outline" size="sm" onClick={addSource}>
                    <Plus className="h-4 w-4 mr-1" />
                    Kaynak Ekle
                  </Button>
                </CardContent>
              </Card>

              <DialogFooter className="gap-2">
                <Button type="button" variant="outline" onClick={closeEditDialog}>
                  İptal
                </Button>
                <Button type="submit" disabled={!hasChanges || updateMutation.isPending}>
                  {updateMutation.isPending ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4 mr-2" />
                  )}
                  {updateMutation.isPending ? 'Kaydediliyor...' : 'Değişiklikleri Kaydet'}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
