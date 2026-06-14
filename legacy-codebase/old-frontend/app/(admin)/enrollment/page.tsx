'use client';

import { useState, useEffect, useCallback } from 'react';
import { enrollmentApi } from '@/lib/api-client';
import type { AvailableCourse } from '@/lib/types';
import CourseList from '@/components/enrollment/CourseList';
import WeeklyScheduleGrid from '@/components/enrollment/WeeklyScheduleGrid';
import Toast from '@/components/enrollment/Toast';

export default function EnrollmentPage() {
  const [selectedCourseIds, setSelectedCourseIds] = useState<string[]>([]);
  const [courseColorMap, setCourseColorMap] = useState<Record<string, number>>({});
  const [nextColorIndex, setNextColorIndex] = useState(0);
  const [availableCourses, setAvailableCourses] = useState<AvailableCourse[]>([]);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState<{ message: string; type: 'error' | 'warning' | 'success' | 'info'; isVisible: boolean }>({
    message: '',
    type: 'warning',
    isVisible: false,
  });

  useEffect(() => {
    fetchCourses();
  }, []);

  const fetchCourses = async () => {
    try {
      setLoading(true);
      const currentSemester = "fall";
      const currentYear = new Date().getFullYear();

      const coursesData = await enrollmentApi
        .get(`available?semester=${currentSemester}`)
        .json<AvailableCourse[]>();

      setAvailableCourses(coursesData || []);
    } catch (err: any) {
      setAvailableCourses([]);
      setToast({
        message: err.message || 'Dersler yüklenemedi',
        type: 'error',
        isVisible: true,
      });
    } finally {
      setLoading(false);
    }
  };

  // Filter courses for selected IDs
  const selectedCourses = availableCourses.filter(course =>
    selectedCourseIds.includes(course.id)
  );

  // Check for conflicts
  const checkForConflicts = useCallback((courseIdToAdd: string) => {
    const newCourse = availableCourses.find(c => c.id === courseIdToAdd);
    if (!newCourse) return [];

    const existingCourses = availableCourses.filter(c =>
      selectedCourseIds.includes(c.id)
    );

    const conflicts: string[] = [];

    newCourse.schedule_sessions.forEach(newSlot => {
      existingCourses.forEach(existingCourse => {
        existingCourse.schedule_sessions.forEach(existingSlot => {
          if (
            newSlot.day === existingSlot.day &&
            newSlot.slot === existingSlot.slot
          ) {
            const conflictStr = `${newCourse.course_code} & ${existingCourse.course_code}`;
            if (!conflicts.includes(conflictStr)) {
              conflicts.push(conflictStr);
            }
          }
        });
      });
    });

    return conflicts;
  }, [selectedCourseIds, availableCourses]);

  // Toggle course selection (add or remove)
  const handleCourseToggle = (courseId: string) => {
    if (selectedCourseIds.includes(courseId)) {
      // Remove course from selection
      setSelectedCourseIds(prev => prev.filter(id => id !== courseId));
    } else {
      // Check for conflicts before adding
      const conflicts = checkForConflicts(courseId);

      if (conflicts.length > 0) {
        setToast({
          message: `Ders çakışması tespit edildi! ${conflicts.join(', ')}`,
          type: 'warning',
          isVisible: true,
        });
      }

      // Add course to selection
      // Assign a color if it doesn't have one yet
      if (!(courseId in courseColorMap)) {
        setCourseColorMap(prevMap => ({
          ...prevMap,
          [courseId]: nextColorIndex
        }));
        setNextColorIndex(prev => prev + 1);
      }
      setSelectedCourseIds(prev => [...prev, courseId]);
    }
  };

  // Remove course from selection
  const handleRemoveCourse = (courseId: string) => {
    setSelectedCourseIds(prev => prev.filter(id => id !== courseId));
  };

  const closeToast = useCallback(() => {
    setToast(prev => ({ ...prev, isVisible: false }));
  }, []);

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-100">
        <p className="text-gray-600">Yükleniyor...</p>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-gray-100">
      {/* Toast Notification */}
      <Toast
        message={toast.message}
        type={toast.type}
        isVisible={toast.isVisible}
        onClose={closeToast}
        duration={5000}
      />

      {/* Left Panel - Course List */}
      <div className="w-80 flex-shrink-0 shadow-lg">
        <CourseList
          courses={availableCourses}
          selectedCourseIds={selectedCourseIds}
          onSelectCourse={handleCourseToggle}
        />
      </div>

      {/* Right Panel - Weekly Schedule Grid */}
      <div className="flex-1 shadow-lg relative">
        <WeeklyScheduleGrid
          sessions={selectedCourses}
          selectedCourseIds={selectedCourseIds}
          courseColorMap={courseColorMap}
          onRemoveCourse={handleRemoveCourse}
        />
      </div>
    </div>
  );
}
