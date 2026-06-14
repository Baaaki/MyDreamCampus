// Export all mock data
export * from './auth';
export * from './staff';
export * from './staff-profile';
export * from './admin-staff';
export * from './students';
export * from './catalog';
export * from './attendance';
export * from './grades';
export * from './meal';

// Combined exports for convenience
import { mockUsers, mockSessions, mockAuthResponse } from './auth';
import { mockStaff } from './staff';
import { mockStaffProfile, mockStaffProfiles } from './staff-profile';
import { mockAdminStaff, mockAdminStaffProfiles } from './admin-staff';
import { mockStudents } from './students';
import { mockCourseCatalog, mockAvailableCourses, mockEnrollmentPrograms, mockFaculties } from './catalog';
import { mockMyAttendanceResponse, mockCourseAttendanceDetails } from './attendance';
import { mockMyGradesResponse, mockTranscriptResponse } from './grades';
import { mockCafeterias, mockMyReservationsResponse, mockQRResponses } from './meal';

export const MockData = {
  // Auth
  users: mockUsers,
  sessions: mockSessions,
  authResponse: mockAuthResponse,

  // Staff & Students
  staff: mockStaff,
  staffProfile: mockStaffProfile,
  staffProfiles: mockStaffProfiles,
  adminStaff: mockAdminStaff,
  adminStaffProfiles: mockAdminStaffProfiles,
  students: mockStudents,

  // Catalog & Enrollment
  courseCatalog: mockCourseCatalog,
  availableCourses: mockAvailableCourses,
  enrollmentPrograms: mockEnrollmentPrograms,
  faculties: mockFaculties,

  // Attendance
  myAttendanceResponse: mockMyAttendanceResponse,
  courseAttendanceDetails: mockCourseAttendanceDetails,

  // Grades
  myGradesResponse: mockMyGradesResponse,
  transcriptResponse: mockTranscriptResponse,

  // Meal
  cafeterias: mockCafeterias,
  myReservationsResponse: mockMyReservationsResponse,
  qrResponses: mockQRResponses,
};
