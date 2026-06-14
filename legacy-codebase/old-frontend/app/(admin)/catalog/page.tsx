"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { mockFaculties } from "@/mock_data/catalog";
import type { CourseCatalog, Department, Faculty } from "@/lib/types";
import { catalogService } from "@/lib/services/catalog-service";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Book,
  Clock,
  GraduationCap,
  Calendar,
  FileText,
  Target,
  List,
  ChevronDown,
  ChevronRight,
  ArrowLeft,
  BookOpen,
  Building2,
  Users,
  Plus
} from "lucide-react";

// Semester info helper
const getSemesterInfo = (semester: number) => {
  const year = Math.ceil(semester / 2);
  const term = semester % 2 === 1 ? "Güz" : "Bahar";
  return { year, term, label: `${year}. Yıl - ${term} Dönemi` };
};

export default function CourseCatalogPage() {
  const router = useRouter();
  const [expandedFaculties, setExpandedFaculties] = useState<string[]>([]);
  const [selectedDepartment, setSelectedDepartment] = useState<{ dept: Department; faculty: Faculty } | null>(null);
  const [selectedCourse, setSelectedCourse] = useState<CourseCatalog | null>(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  // API state
  const [departmentCourses, setDepartmentCourses] = useState<CourseCatalog[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [courseCounts, setCourseCounts] = useState<Record<string, number>>({});

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

  // Fetch course count for a department (for display in accordion)
  const fetchCourseCounts = useCallback(async () => {
    try {
      const { courses } = await catalogService.listCourses({ limit: 100 }); // Backend max is 100
      const counts: Record<string, number> = {};
      courses.forEach(course => {
        counts[course.department] = (counts[course.department] || 0) + 1;
      });
      setCourseCounts(counts);
    } catch (err) {
      console.error('Failed to fetch course counts:', err);
    }
  }, []);

  // Load course counts on mount
  useEffect(() => {
    fetchCourseCounts();
  }, [fetchCourseCounts]);

  // Load courses when department changes
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

  const handleCourseClick = async (course: CourseCatalog) => {
    setSelectedCourse(course); // Show basic info immediately
    setIsDetailOpen(true);

    // Fetch full course details for complete information
    try {
      const fullCourse = await catalogService.getCourseByCode(course.course_code);
      setSelectedCourse(fullCourse);
    } catch (err) {
      console.error('Failed to fetch course details:', err);
      // Keep showing basic info if detailed fetch fails
    }
  };

  const handleDepartmentClick = (dept: Department, faculty: Faculty) => {
    setSelectedDepartment({ dept, faculty });
  };

  // Faculty & Department List View (Accordion)
  if (!selectedDepartment) {
    return (
      <div className="min-h-screen bg-gray-50 py-8">
        <div className="max-w-4xl mx-auto px-4">
          <div className="bg-white rounded-lg shadow-md p-6">
            <div className="flex items-center justify-between mb-8">
              <div className="text-center flex-1">
                <h1 className="text-3xl font-bold text-gray-900 mb-2">Ders Kataloğu</h1>
                <p className="text-gray-600">Fakülte ve bölüm seçerek ders programını görüntüleyebilirsiniz</p>
              </div>
              <Button
                onClick={() => router.push('/catalog/add')}
                className="bg-indigo-600 hover:bg-indigo-700"
              >
                <Plus className="h-4 w-4 mr-2" />
                Yeni Ders Ekle
              </Button>
            </div>

            {/* Faculty Accordion */}
            <div className="space-y-2">
              {mockFaculties.map((faculty) => {
                const isExpanded = expandedFaculties.includes(faculty.id);
                return (
                  <div key={faculty.id} className="border rounded-lg overflow-hidden">
                    {/* Faculty Header */}
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

                    {/* Departments List */}
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

            {/* Stats */}
            <div className="mt-8 pt-6 border-t grid grid-cols-2 gap-4 text-center">
              <div className="p-4 bg-indigo-50 rounded-lg">
                <div className="text-2xl font-bold text-indigo-600">{mockFaculties.length}</div>
                <div className="text-sm text-gray-600">Fakülte</div>
              </div>
              <div className="p-4 bg-green-50 rounded-lg">
                <div className="text-2xl font-bold text-green-600">
                  {mockFaculties.reduce((sum, f) => sum + f.departments.length, 0)}
                </div>
                <div className="text-sm text-gray-600">Bölüm</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Department Detail View (Semester-based course listing)
  const groupedCourses = groupCoursesBySemester(departmentCourses);

  // Loading state
  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 py-8">
        <div className="max-w-7xl mx-auto px-4">
          <div className="bg-white rounded-lg shadow-md p-6 mb-6">
            <button
              onClick={() => setSelectedDepartment(null)}
              className="flex items-center gap-2 text-indigo-600 hover:text-indigo-800 mb-4 transition-colors"
            >
              <ArrowLeft className="h-5 w-5" />
              <span>Tüm Fakülteler</span>
            </button>
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600"></div>
              <span className="ml-4 text-gray-600">Dersler yükleniyor...</span>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 py-8">
        <div className="max-w-7xl mx-auto px-4">
          <div className="bg-white rounded-lg shadow-md p-6 mb-6">
            <button
              onClick={() => setSelectedDepartment(null)}
              className="flex items-center gap-2 text-indigo-600 hover:text-indigo-800 mb-4 transition-colors"
            >
              <ArrowLeft className="h-5 w-5" />
              <span>Tüm Fakülteler</span>
            </button>
            <div className="text-center py-12">
              <div className="text-red-500 mb-4">
                <BookOpen className="h-16 w-16 mx-auto text-red-300" />
              </div>
              <h2 className="text-xl font-semibold text-gray-700 mb-2">Hata Oluştu</h2>
              <p className="text-gray-500 mb-4">{error}</p>
              <Button
                onClick={() => fetchDepartmentCourses(selectedDepartment.dept.name)}
                variant="outline"
              >
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

          <div className="flex items-center gap-4">
            <div className="w-16 h-16 bg-indigo-100 rounded-xl flex items-center justify-center">
              <GraduationCap className="h-8 w-8 text-indigo-600" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{selectedDepartment.dept.name}</h1>
              <p className="text-gray-600">{selectedDepartment.faculty.name}</p>
              <p className="text-sm text-gray-500 mt-1">{departmentCourses.length} ders</p>
            </div>
          </div>
        </div>

        {/* Course content check */}
        {departmentCourses.length === 0 ? (
          <div className="bg-white rounded-lg shadow-md p-8 text-center">
            <BookOpen className="h-16 w-16 text-gray-300 mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-gray-700 mb-2">Ders Bilgisi Bulunamadı</h2>
            <p className="text-gray-500">Bu bölüm için henüz ders bilgisi eklenmemiş.</p>
          </div>
        ) : (
          /* Semester Grid - 4 years x 2 semesters */
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
                                  </tr>
                                </thead>
                                <tbody>
                                  {semesterCourses.mandatory.map((course) => (
                                    <tr
                                      key={course.id}
                                      className="border-b border-gray-100 hover:bg-red-50 cursor-pointer transition-colors"
                                      onClick={() => handleCourseClick(course)}
                                    >
                                      <td className="px-3 py-2 font-medium text-indigo-600">{course.course_code}</td>
                                      <td className="px-3 py-2 text-gray-800">{course.name}</td>
                                      <td className="px-3 py-2 text-center text-gray-600">{course.theoretical_hours}</td>
                                      <td className="px-3 py-2 text-center font-medium text-gray-800">{course.ects || course.credits}</td>
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
                                    <th className="px-3 py-2 font-medium text-gray-700 text-center">U</th>
                                    <th className="px-3 py-2 font-medium text-gray-700 text-center">AKTS</th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {semesterCourses.elective.map((course) => (
                                    <tr
                                      key={course.id}
                                      className="border-b border-gray-100 hover:bg-green-50 cursor-pointer transition-colors"
                                      onClick={() => handleCourseClick(course)}
                                    >
                                      <td className="px-3 py-2 font-medium text-indigo-600">{course.course_code}</td>
                                      <td className="px-3 py-2 text-gray-800">{course.name}</td>
                                      <td className="px-3 py-2 text-center text-gray-600">{course.theoretical_hours}</td>
                                      <td className="px-3 py-2 text-center text-gray-600">{course.lab_hours}</td>
                                      <td className="px-3 py-2 text-center font-medium text-gray-800">{course.ects || course.credits}</td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          </div>
                        )}

                        {/* Empty state for semester */}
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

      {/* Course Detail Dialog */}
      <Dialog open={isDetailOpen} onOpenChange={setIsDetailOpen}>
        <DialogContent className="sm:max-w-3xl md:max-w-4xl lg:max-w-5xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="text-xl">
              {selectedCourse?.course_code} - {selectedCourse?.name}
            </DialogTitle>
          </DialogHeader>

          {selectedCourse && (
            <div className="space-y-6 flex-1">
              {/* Badges */}
              <div className="flex gap-2 flex-wrap">
                <Badge variant={selectedCourse.course_type === 'mandatory' ? 'default' : 'secondary'}>
                  {selectedCourse.course_type === 'mandatory' ? 'Zorunlu' : 'Seçmeli'}
                </Badge>
                <Badge variant="outline">{selectedCourse.class_level}. Sınıf</Badge>
                {selectedCourse.semester && (
                  <Badge variant="outline">{selectedCourse.semester}. Dönem</Badge>
                )}
                {selectedCourse.education_level && (
                  <Badge variant="outline">{selectedCourse.education_level}</Badge>
                )}
                {selectedCourse.language && (
                  <Badge variant="outline">{selectedCourse.language}</Badge>
                )}
              </div>

              {/* Basic Info Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                <div className="flex items-center gap-2 bg-gray-50 p-3 rounded-lg">
                  <Book className="h-5 w-5 text-indigo-500 flex-shrink-0" />
                  <span className="text-gray-600">Fakülte:</span>
                  <span className="font-medium text-gray-900">{selectedCourse.faculty}</span>
                </div>
                <div className="flex items-center gap-2 bg-gray-50 p-3 rounded-lg">
                  <GraduationCap className="h-5 w-5 text-indigo-500 flex-shrink-0" />
                  <span className="text-gray-600">Bölüm:</span>
                  <span className="font-medium text-gray-900">{selectedCourse.department}</span>
                </div>
                {selectedCourse.offering_unit && (
                  <div className="flex items-center gap-2 bg-gray-50 p-3 rounded-lg md:col-span-2">
                    <BookOpen className="h-5 w-5 text-indigo-500 flex-shrink-0" />
                    <span className="text-gray-600">Dersi Veren Birim:</span>
                    <span className="font-medium text-gray-900">{selectedCourse.offering_unit}</span>
                  </div>
                )}
                {selectedCourse.teaching_type && (
                  <div className="flex items-center gap-2 bg-gray-50 p-3 rounded-lg">
                    <FileText className="h-5 w-5 text-indigo-500 flex-shrink-0" />
                    <span className="text-gray-600">Öğretim Türü:</span>
                    <span className="font-medium text-gray-900">{selectedCourse.teaching_type}</span>
                  </div>
                )}
              </div>

              {/* Credits & Hours */}
              <div className="bg-gradient-to-r from-indigo-50 to-blue-50 rounded-xl p-5">
                <h4 className="font-bold text-gray-900 mb-3 flex items-center gap-2 text-base">
                  <Clock className="h-5 w-5 text-indigo-600" />
                  Kredi ve Saat Bilgileri
                </h4>
                <div className="grid grid-cols-5 gap-3">
                  <div className="text-center p-3 bg-white rounded-lg border border-indigo-100">
                    <div className="text-2xl font-bold text-indigo-600">{selectedCourse.theoretical_hours}</div>
                    <div className="text-xs text-gray-600 mt-1">Teorik (D)</div>
                  </div>
                  <div className="text-center p-3 bg-white rounded-lg border border-blue-100">
                    <div className="text-2xl font-bold text-blue-600">{selectedCourse.lab_hours}</div>
                    <div className="text-xs text-gray-600 mt-1">Uygulama (U)</div>
                  </div>
                  <div className="text-center p-3 bg-white rounded-lg border border-cyan-100">
                    <div className="text-2xl font-bold text-cyan-600">{selectedCourse.lab_hours || 0}</div>
                    <div className="text-xs text-gray-600 mt-1">Lab (L)</div>
                  </div>
                  <div className="text-center p-3 bg-white rounded-lg border border-purple-100">
                    <div className="text-2xl font-bold text-purple-600">{selectedCourse.credits}</div>
                    <div className="text-xs text-gray-600 mt-1">Kredi</div>
                  </div>
                  <div className="text-center p-3 bg-white rounded-lg border border-green-100">
                    <div className="text-2xl font-bold text-green-600">{selectedCourse.ects || selectedCourse.credits}</div>
                    <div className="text-xs text-gray-600 mt-1">AKTS</div>
                  </div>
                </div>
              </div>

              {/* Coordinator */}
              {selectedCourse.coordinator && (
                <div className="bg-blue-50 border border-blue-200 rounded-xl p-5">
                  <h4 className="font-bold text-gray-900 mb-3 flex items-center gap-2 text-base">
                    <GraduationCap className="h-5 w-5 text-blue-600" />
                    Ders Koordinatörü
                  </h4>
                  <div className="space-y-2 text-sm">
                    <p className="font-semibold text-gray-900">
                      {selectedCourse.coordinator.title} {selectedCourse.coordinator.name}
                    </p>
                    {selectedCourse.coordinator.email && (
                      <p className="text-gray-600">
                        <span className="font-medium">E-posta:</span> {selectedCourse.coordinator.email}
                      </p>
                    )}
                    {selectedCourse.coordinator.phone && (
                      <p className="text-gray-600">
                        <span className="font-medium">Telefon:</span> {selectedCourse.coordinator.phone}
                      </p>
                    )}
                    {selectedCourse.coordinator.office && (
                      <p className="text-gray-600">
                        <span className="font-medium">Ofis:</span> {selectedCourse.coordinator.office}
                      </p>
                    )}
                  </div>
                </div>
              )}

              {/* Purpose */}
              {selectedCourse.purpose && (
                <div className="bg-white border rounded-xl p-5">
                  <h4 className="font-bold text-gray-900 mb-2 flex items-center gap-2 text-base">
                    <Target className="h-5 w-5 text-purple-500" />
                    Dersin Amacı
                  </h4>
                  <p className="text-gray-700 text-sm leading-relaxed">
                    {selectedCourse.purpose}
                  </p>
                </div>
              )}

              {/* Learning Outcomes List */}
              {selectedCourse.learning_outcomes_list && selectedCourse.learning_outcomes_list.length > 0 && (
                <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-5">
                  <h4 className="font-bold text-gray-900 mb-3 flex items-center gap-2 text-base">
                    <Target className="h-5 w-5 text-emerald-600" />
                    Öğrenme Kazanımları
                  </h4>
                  <ol className="space-y-2 text-sm">
                    {selectedCourse.learning_outcomes_list.map((outcome, index) => (
                      <li key={index} className="flex gap-3 text-gray-700">
                        <span className="flex-shrink-0 w-6 h-6 bg-emerald-500 text-white rounded-full flex items-center justify-center text-xs font-bold">
                          {index + 1}
                        </span>
                        <span className="leading-relaxed">{outcome}</span>
                      </li>
                    ))}
                  </ol>
                </div>
              )}

              {/* Weekly Topics */}
              {selectedCourse.weekly_topics && selectedCourse.weekly_topics.length > 0 && (
                <div className="bg-white border rounded-xl p-5">
                  <h4 className="font-bold text-gray-900 mb-3 flex items-center gap-2 text-base">
                    <Calendar className="h-5 w-5 text-cyan-500" />
                    Ders İçeriği (Haftalık)
                  </h4>
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="bg-gray-50">
                          <th className="px-3 py-2 text-left font-medium text-gray-700 w-20">Hafta</th>
                          <th className="px-3 py-2 text-left font-medium text-gray-700">Konu</th>
                        </tr>
                      </thead>
                      <tbody>
                        {selectedCourse.weekly_topics.map((topic) => (
                          <tr key={topic.week} className="border-b border-gray-100">
                            <td className="px-3 py-2 font-medium text-indigo-600">{topic.week}</td>
                            <td className="px-3 py-2 text-gray-700">{topic.topic}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {/* Recommended Sources */}
              {selectedCourse.recommended_sources && selectedCourse.recommended_sources.length > 0 && (
                <div className="bg-amber-50 border border-amber-200 rounded-xl p-5">
                  <h4 className="font-bold text-gray-900 mb-3 flex items-center gap-2 text-base">
                    <BookOpen className="h-5 w-5 text-amber-600" />
                    Önerilen Kaynaklar
                  </h4>
                  <ul className="space-y-2 text-sm">
                    {selectedCourse.recommended_sources.map((source, index) => (
                      <li key={index} className="flex gap-2 text-gray-700">
                        <span className="text-amber-500">•</span>
                        <span>{source}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Prerequisites */}
              {selectedCourse.prerequisites && selectedCourse.prerequisites.length > 0 && (
                <div className="bg-orange-50 border border-orange-200 rounded-xl p-5">
                  <h4 className="font-bold text-gray-900 mb-3 flex items-center gap-2 text-base">
                    <List className="h-5 w-5 text-orange-500" />
                    Ön Koşullar
                  </h4>
                  <div className="space-y-2">
                    {selectedCourse.prerequisites.map((prereq) => (
                      <div
                        key={prereq.id}
                        className="flex items-center gap-3 p-3 bg-white rounded-lg border border-orange-200 text-sm"
                      >
                        <Badge variant="outline" className="bg-orange-100">
                          {prereq.course_code}
                        </Badge>
                        <span className="text-gray-800 font-medium">{prereq.course_name}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* No Prerequisites */}
              {(!selectedCourse.prerequisites || selectedCourse.prerequisites.length === 0) && (
                <div className="bg-gray-50 border rounded-xl p-5">
                  <h4 className="font-bold text-gray-900 mb-2 flex items-center gap-2 text-base">
                    <List className="h-5 w-5 text-gray-500" />
                    Ön Koşullar
                  </h4>
                  <p className="text-gray-500 text-sm">Yok</p>
                </div>
              )}

              {/* Dates */}
              <div className="border-t pt-4 text-xs text-gray-400 flex justify-between">
                <span>Oluşturulma: {new Date(selectedCourse.created_at).toLocaleDateString('tr-TR')}</span>
                <span>Güncelleme: {new Date(selectedCourse.updated_at).toLocaleDateString('tr-TR')}</span>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
