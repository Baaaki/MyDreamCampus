'use client';

import { useState } from 'react';
import { AvailableCourse } from '@/lib/types';

interface CourseListProps {
  courses: AvailableCourse[];
  selectedCourseIds: string[];
  onSelectCourse: (courseId: string) => void;
}

type YearGroup = '1' | '2' | '3' | '4' | 'secmeli';

interface AccordionSection {
  key: YearGroup;
  title: string;
  courses: AvailableCourse[];
}

// Extract branch/section from course code (e.g., "BİL 1007-1" -> "1")
function getCourseBranch(courseCode: string): string | null {
  const match = courseCode.match(/-(\d+)$/);
  return match ? match[1] : null;
}

// Extract year from course code
function getCourseYear(courseCode: string): YearGroup {
  // FSH codes are electives
  if (courseCode.toUpperCase().startsWith('FSH')) {
    return 'secmeli';
  }

  // Find the first digit in the course code (after the letters)
  const match = courseCode.match(/\d/);
  if (match) {
    const firstDigit = match[0];
    if (firstDigit === '1') return '1';
    if (firstDigit === '2') return '2';
    if (firstDigit === '3') return '3';
    if (firstDigit === '4') return '4';
  }

  // Default to elective if no match
  return 'secmeli';
}

// Group courses by year
function groupCoursesByYear(courses: AvailableCourse[]): AccordionSection[] {
  const groups: Record<YearGroup, AvailableCourse[]> = {
    '1': [],
    '2': [],
    '3': [],
    '4': [],
    'secmeli': [],
  };

  courses.forEach(course => {
    const year = getCourseYear(course.course_code);
    groups[year].push(course);
  });

  return [
    { key: '1', title: '1. Sınıf Dersleri', courses: groups['1'] },
    { key: '2', title: '2. Sınıf Dersleri', courses: groups['2'] },
    { key: '3', title: '3. Sınıf Dersleri', courses: groups['3'] },
    { key: '4', title: '4. Sınıf Dersleri', courses: groups['4'] },
    { key: 'secmeli', title: 'Seçmeli Dersler', courses: groups['secmeli'] },
  ].filter(section => section.courses.length > 0);
}

export default function CourseList({ courses, selectedCourseIds, onSelectCourse }: CourseListProps) {
  const [expandedSections, setExpandedSections] = useState<Set<YearGroup>>(new Set());

  const sections = groupCoursesByYear(courses);

  const toggleSection = (key: YearGroup) => {
    setExpandedSections(prev => {
      const newSet = new Set(prev);
      if (newSet.has(key)) {
        newSet.delete(key);
      } else {
        newSet.add(key);
      }
      return newSet;
    });
  };

  // Count selected courses per section
  const getSelectedCount = (sectionCourses: AvailableCourse[]) => {
    return sectionCourses.filter(c => selectedCourseIds.includes(c.id)).length;
  };

  return (
    <div className="h-full flex flex-col bg-white border-r border-gray-200">
      <div className="px-3 py-2 border-b border-gray-200 bg-gray-50">
        <h2 className="text-lg font-bold text-gray-800">Açılan Dersler</h2>
        <p className="text-xs text-gray-600">
          {courses.length} ders • {selectedCourseIds.length} seçili
        </p>
      </div>

      <div className="flex-1 overflow-y-auto">
        {sections.map((section) => {
          const isExpanded = expandedSections.has(section.key);
          const selectedCount = getSelectedCount(section.courses);

          return (
            <div key={section.key} className="border-b border-gray-200">
              {/* Accordion Header */}
              <button
                onClick={() => toggleSection(section.key)}
                className="w-full px-3 py-2 flex items-center justify-between bg-gray-100 hover:bg-gray-200 transition-colors"
              >
                <div className="flex items-center gap-2">
                  <svg
                    className={`w-4 h-4 text-gray-600 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                  <span className="font-semibold text-sm text-gray-800">{section.title}</span>
                </div>
                <div className="flex items-center gap-2">
                  {selectedCount > 0 && (
                    <span className="text-xs bg-blue-500 text-white px-1.5 py-0.5 rounded-full">
                      {selectedCount}
                    </span>
                  )}
                  <span className="text-xs text-gray-500">{section.courses.length}</span>
                </div>
              </button>

              {/* Accordion Content */}
              <div
                className={`overflow-hidden transition-all duration-200 ${
                  isExpanded ? 'max-h-[2000px]' : 'max-h-0'
                }`}
              >
                {section.courses.map((course) => {
                  const isSelected = selectedCourseIds.includes(course.id);
                  const branch = getCourseBranch(course.course_code);
                  return (
                    <div
                      key={course.id}
                      onClick={() => onSelectCourse(course.id)}
                      className={`px-3 py-2 border-b border-gray-100 cursor-pointer transition-all duration-200 hover:bg-blue-50 ${
                        isSelected
                          ? 'bg-blue-100 border-l-4 border-l-blue-600'
                          : 'hover:border-l-4 hover:border-l-blue-300'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <h3 className="font-bold text-sm text-gray-900">{course.course_code}</h3>
                        {branch && (
                          <span className="text-xs bg-purple-100 text-purple-700 px-1.5 py-0.5 rounded-full">
                            {branch}. Şube
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-1 text-xs text-gray-600 mt-1">
                        <svg className="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                        </svg>
                        <span className="truncate">{course.instructor}</span>
                      </div>
                      <div className="flex items-center gap-1 text-xs text-gray-500">
                        <span className="truncate">
                          Kontenjan: {course.current_enrollment}/{course.max_capacity} • Boş: {course.available_seats}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
