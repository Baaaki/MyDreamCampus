import { MockData } from '@/mock_data';

// Simulate network delay
const delay = (ms: number = 300) => new Promise(resolve => setTimeout(resolve, ms));

// Mock API Response type that mimics ky's response
type MockResponse<T> = {
  json: () => Promise<T>;
};

// Mock API Client
class MockApiClient {
  private prefixUrl?: string;

  get(url: string, options?: any): MockResponse<any> {
    // Prepend prefixUrl if exists
    let fullUrl = this.prefixUrl ? `${this.prefixUrl}/${url}` : url;

    // Build full URL with search params if provided
    if (options?.searchParams) {
      const params = new URLSearchParams(options.searchParams);
      fullUrl = `${fullUrl}?${params.toString()}`;
    }

    console.log('[Mock API] GET', fullUrl);

    // Parse URL to extract route
    const cleanUrl = fullUrl.replace(/^\//, '').replace(/^api\//, '');
    console.log('[Mock API] cleanUrl:', cleanUrl, 'prefixUrl:', this.prefixUrl);

    return {
      json: async () => {
        await delay();

        // Auth endpoints
        if (cleanUrl === 'sessions') {
          return MockData.sessions;
        }

        // Staff endpoints
        const isStaffEndpoint =
          cleanUrl === 'staff' ||
          cleanUrl === 'staff/' ||
          cleanUrl.startsWith('staff?') ||
          cleanUrl.startsWith('staff/?') ||
          (this.prefixUrl?.includes('staff') && (cleanUrl === '' || cleanUrl.startsWith('?')));

        if (isStaffEndpoint) {
          console.log('[Mock API] Staff endpoint matched:', cleanUrl, 'prefixUrl:', this.prefixUrl);

          // Parse query params for filtering
          let urlForParsing = cleanUrl;
          if (cleanUrl === '' || cleanUrl === 'staff' || cleanUrl === 'staff/') {
            urlForParsing = '?';
          } else if (!cleanUrl.includes('?')) {
            urlForParsing = cleanUrl + '?';
          }

          const urlObj = new URL('http://dummy.com/' + urlForParsing);
          const params = urlObj.searchParams;
          const department = params.get('department');
          const page = parseInt(params.get('page') || '1', 10);
          const limit = parseInt(params.get('limit') || '10', 10);

          let staffList = MockData.staff;
          if (department) {
            staffList = staffList.filter(s => s.department === department);
          }

          console.log('[Mock API] Staff list:', staffList.length, 'total');

          // Return paginated response matching backend structure
          return {
            data: staffList,
            pagination: {
              page: page,
              limit: limit,
              total: staffList.length,
              total_pages: Math.ceil(staffList.length / limit),
            },
          };
        }
        // Staff profile by ID endpoint - staff/profile/:id
        if (cleanUrl.match(/^staff\/profile\/[\w-]+$/) || (cleanUrl.match(/^profile\/[\w-]+$/) && this.prefixUrl?.includes('staff'))) {
          const id = cleanUrl.split('/').pop();
          console.log('[Mock API] Staff profile by ID matched:', id);
          const profile = MockData.staffProfiles[id || ''];
          return profile || null;
        }
        // Staff profile endpoint (default) - must be before staff/:id check
        if (cleanUrl === 'staff/profile' || (cleanUrl === 'profile' && this.prefixUrl?.includes('staff'))) {
          console.log('[Mock API] Staff profile matched');
          return MockData.staffProfile;
        }

        // Admin staff list endpoint
        if (cleanUrl === 'admin-staff' || cleanUrl === 'admin-staff/') {
          console.log('[Mock API] Admin staff list matched');
          return MockData.adminStaff;
        }
        // Admin staff profile by ID endpoint
        if (cleanUrl.match(/^admin-staff\/profile\/[\w-]+$/)) {
          const id = cleanUrl.split('/').pop();
          console.log('[Mock API] Admin staff profile by ID matched:', id);
          const profile = MockData.adminStaffProfiles[id || ''];
          return profile || null;
        }

        if (cleanUrl.match(/^staff\/[\w-]+$/)) {
          const id = cleanUrl.replace('staff/', '');
          const staff = MockData.staff.find(s => s.id === id);
          return staff || null;
        }
        if (cleanUrl.match(/^students\/[\w-]+$/)) {
          const id = cleanUrl.replace('students/', '');
          const student = MockData.students.find(s => s.id === id);
          return student || null;
        }

        // Student endpoints (handle both with and without 'students' prefix)
        const isStudentEndpoint =
          cleanUrl === 'students' ||
          cleanUrl === 'students/' ||
          cleanUrl.startsWith('students?') ||
          cleanUrl.startsWith('students/?') ||
          (this.prefixUrl?.includes('students') && (cleanUrl === '' || cleanUrl.startsWith('?')));

        if (isStudentEndpoint) {
          console.log('[Mock API] Student endpoint matched:', cleanUrl, 'prefixUrl:', this.prefixUrl);

          // Parse query params for filtering
          let urlForParsing = cleanUrl;
          if (cleanUrl === '') {
            urlForParsing = '?';
          } else if (!cleanUrl.includes('?')) {
            urlForParsing = cleanUrl + '?';
          }

          const urlObj = new URL('http://dummy.com/' + urlForParsing);
          const params = urlObj.searchParams;
          const page = parseInt(params.get('page') || '1', 10);
          const limit = parseInt(params.get('limit') || '10', 10);

          let studentList = MockData.students;

          // Apply pagination
          const startIndex = (page - 1) * limit;
          const endIndex = startIndex + limit;
          const paginatedData = studentList.slice(startIndex, endIndex);

          console.log('[Mock API] Students:', studentList.length, 'total,', paginatedData.length, 'on page', page);

          // Return paginated response matching backend structure
          return {
            data: paginatedData,
            pagination: {
              page: page,
              limit: limit,
              total: studentList.length,
              total_pages: Math.ceil(studentList.length / limit),
            },
          };
        }

        // Orphaned students (no advisor)
        const isOrphanedEndpoint =
          cleanUrl === 'orphaned' ||
          cleanUrl.startsWith('orphaned?') ||
          cleanUrl === 'students/orphaned' ||
          cleanUrl.startsWith('students/orphaned?') ||
          (this.prefixUrl?.includes('students') && (cleanUrl === 'orphaned' || cleanUrl.startsWith('orphaned?')));

        if (isOrphanedEndpoint) {
          // Parse query params for pagination
          let urlForParsing = cleanUrl;
          if (!cleanUrl.includes('?')) {
            urlForParsing = cleanUrl + '?';
          }

          const urlObj = new URL('http://dummy.com/' + urlForParsing);
          const params = urlObj.searchParams;
          const page = parseInt(params.get('page') || '1', 10);
          const limit = parseInt(params.get('limit') || '10', 10);

          const orphanedStudents = MockData.students.filter(s => !s.advisor_id);
          const startIndex = (page - 1) * limit;
          const endIndex = startIndex + limit;
          const paginatedData = orphanedStudents.slice(startIndex, endIndex);

          console.log('[Mock API] Orphaned students:', orphanedStudents.length, 'total,', paginatedData.length, 'on page', page);

          return {
            data: paginatedData,
            pagination: {
              page: page,
              limit: limit,
              total: orphanedStudents.length,
              total_pages: Math.ceil(orphanedStudents.length / limit),
            },
          };
        }

        // Get advisor's students (advisees)
        if (cleanUrl.match(/^advisors\/[\w-]+\/advisees/) || cleanUrl.match(/^students\/advisors\/[\w-]+\/advisees/)) {
          const parts = cleanUrl.split('/');
          const advisorIdIndex = parts.indexOf('advisors') + 1;
          const advisorId = parts[advisorIdIndex];

          const urlObj = new URL('http://dummy.com/' + cleanUrl);
          const params = urlObj.searchParams;
          const page = parseInt(params.get('page') || '1', 10);
          const limit = parseInt(params.get('limit') || '10', 10);

          const advisorStudents = MockData.students.filter(s => s.advisor_id === advisorId);
          const startIndex = (page - 1) * limit;
          const endIndex = startIndex + limit;
          const paginatedData = advisorStudents.slice(startIndex, endIndex);

          console.log('[Mock API] Advisor students for', advisorId, ':', advisorStudents.length, 'total,', paginatedData.length, 'on page', page);

          return {
            data: paginatedData,
            pagination: {
              page: page,
              limit: limit,
              total: advisorStudents.length,
              total_pages: Math.ceil(advisorStudents.length / limit),
            },
          };
        }
        if (cleanUrl === 'my' && this.prefixUrl?.includes('students')) {
          return MockData.students[0];
        }

        // Catalog endpoints
        const isCatalogEndpoint =
          cleanUrl === 'catalog' ||
          cleanUrl === 'courses' ||
          cleanUrl === 'catalog/courses' ||
          cleanUrl.startsWith('courses?') ||
          cleanUrl.startsWith('catalog/courses?') ||
          (this.prefixUrl?.includes('catalog') && (cleanUrl === '' || cleanUrl === 'courses' || cleanUrl.startsWith('courses?') || cleanUrl.startsWith('?')));

        if (isCatalogEndpoint) {
          console.log('[Mock API] Catalog endpoint matched:', cleanUrl, 'prefixUrl:', this.prefixUrl);

          // Parse query params for filtering
          let urlForParsing = cleanUrl;
          if (!cleanUrl.includes('?')) {
            urlForParsing = cleanUrl + '?';
          }

          const urlObj = new URL('http://dummy.com/' + urlForParsing);
          const params = urlObj.searchParams;
          const faculty = params.get('faculty');
          const department = params.get('department');
          const courseType = params.get('course_type');

          let filteredCourses = MockData.courseCatalog;
          if (faculty) {
            filteredCourses = filteredCourses.filter(c => c.faculty === faculty);
          }
          if (department) {
            filteredCourses = filteredCourses.filter(c => c.department === department);
          }
          if (courseType) {
            filteredCourses = filteredCourses.filter(c => c.course_type === courseType);
          }

          console.log('[Mock API] Catalog courses:', filteredCourses.length, 'total');
          return filteredCourses;
        }

        // Enrollment endpoints
        if ((cleanUrl === 'available' || cleanUrl.startsWith('available?')) && this.prefixUrl?.includes('enrollment')) {
          return MockData.availableCourses;
        }
        if (cleanUrl === 'my' && this.prefixUrl?.includes('enrollment')) {
          return MockData.enrollmentPrograms;
        }

        // Attendance endpoints
        if (cleanUrl === 'my' && this.prefixUrl?.includes('attendance')) {
          return MockData.myAttendanceResponse;
        }

        // Grades endpoints
        if (cleanUrl === 'my' && this.prefixUrl?.includes('grades')) {
          return MockData.myGradesResponse;
        }
        if (cleanUrl === 'transcript/my') {
          return MockData.transcriptResponse;
        }

        // Meal endpoints
        if (cleanUrl === 'cafeterias') {
          return MockData.cafeterias.filter(c => c.is_active);
        }
        if (cleanUrl === 'reservations/my') {
          return MockData.myReservationsResponse;
        }
        if (cleanUrl.match(/^qr\//)) {
          return MockData.qrResponses[0];
        }

        console.warn('[Mock API] Unhandled GET:', cleanUrl);
        return null;
      }
    };
  }

  post(url: string, options?: any): MockResponse<any> {
    const fullUrl = this.prefixUrl ? `${this.prefixUrl}/${url}` : url;
    console.log('[Mock API] POST', fullUrl, options);

    return {
      json: async () => {
        await delay();

        const cleanUrl = fullUrl.replace(/^\//, '').replace(/^api\//, '');

        // Auth endpoints
        if (cleanUrl === 'login') {
          return MockData.authResponse;
        }
        if (cleanUrl === 'logout') {
          return { success: true };
        }
        if (cleanUrl === 'change-password') {
          return { success: true };
        }

        // Enrollment endpoints
        if (cleanUrl === 'enroll') {
          return { success: true, message: 'Kayıt başarılı' };
        }

        // Attendance endpoints
        if (cleanUrl === 'mark') {
          return { success: true, status: 'present' };
        }
        if (cleanUrl.match(/^sessions\/create/)) {
          return {
            id: 'new-session-' + Date.now(),
            qr_secret: 'QR-' + Date.now(),
            expires_at: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
          };
        }

        // Meal endpoints
        if (cleanUrl === 'reservations') {
          return {
            id: 'new-reservation-' + Date.now(),
            success: true,
            message: 'Rezervasyon oluşturuldu',
          };
        }

        // Staff endpoints
        if (cleanUrl === 'staff') {
          return {
            id: 'new-staff-' + Date.now(),
            success: true,
            message: 'Staff created successfully',
          };
        }

        // Student endpoints (handle both with and without 'students' prefix)
        if (cleanUrl === 'students' || (cleanUrl === '' && this.prefixUrl?.includes('students'))) {
          const newStudent = {
            id: 'new-student-' + Date.now(),
            ...options?.json,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          };

          // Add to mock data
          MockData.students.push(newStudent);

          return {
            id: newStudent.id,
            success: true,
            message: 'Student created successfully',
            data: newStudent,
          };
        }

        // Bulk assign advisor (handle both old and new URL formats)
        if (cleanUrl === 'students/advisors/bulk-assign' || cleanUrl === 'advisors/bulk-assign') {
          const { student_ids, advisor_id } = options?.json || {};

          // Update mock data
          if (student_ids && advisor_id) {
            student_ids.forEach((studentId: string) => {
              const student = MockData.students.find(s => s.id === studentId);
              if (student) {
                student.advisor_id = advisor_id;
                // Also populate advisor info
                const advisor = MockData.staff.find(s => s.id === advisor_id);
                if (advisor) {
                  student.advisor = {
                    id: advisor.id,
                    first_name: advisor.first_name,
                    last_name: advisor.last_name,
                    email: advisor.email,
                  };
                }
              }
            });
          }

          return {
            success: true,
            message: 'Toplu danışman ataması başarılı',
            updated_count: student_ids?.length || 0,
          };
        }

        console.warn('[Mock API] Unhandled POST:', cleanUrl);
        return { success: false };
      }
    };
  }

  patch(url: string, options?: any): MockResponse<any> {
    const fullUrl = this.prefixUrl ? `${this.prefixUrl}/${url}` : url;
    console.log('[Mock API] PATCH', fullUrl, options);

    return {
      json: async () => {
        await delay();

        const cleanUrl = fullUrl.replace(/^\//, '').replace(/^api\//, '');

        // Update student advisor
        if (cleanUrl.match(/^students\/[\w-]+$/)) {
          const studentId = cleanUrl.split('/').pop();
          const { advisor_id } = options?.json || {};

          const student = MockData.students.find(s => s.id === studentId);
          if (student && advisor_id) {
            student.advisor_id = advisor_id;
            // Also populate advisor info
            const advisor = MockData.staff.find(s => s.id === advisor_id);
            if (advisor) {
              student.advisor = {
                id: advisor.id,
                first_name: advisor.first_name,
                last_name: advisor.last_name,
                email: advisor.email,
              };
            }

            return {
              success: true,
              message: 'Danışman başarıyla atandı',
              data: student
            };
          }
        }

        return { success: true, message: 'Güncelleme başarılı' };
      }
    };
  }

  put(url: string, options?: any): MockResponse<any> {
    const fullUrl = this.prefixUrl ? `${this.prefixUrl}/${url}` : url;
    console.log('[Mock API] PUT', fullUrl, options);

    return {
      json: async () => {
        await delay();

        const cleanUrl = fullUrl.replace(/^\//, '').replace(/^api\//, '');

        // Update student - handle both full path and prefixed URL
        const studentMatch = cleanUrl.match(/^students\/([\w-]+)$/);
        const isStudentUpdate = studentMatch || (this.prefixUrl?.includes('students') && url && !url.includes('/'));

        if (isStudentUpdate) {
          const studentId = studentMatch ? studentMatch[1] : url;
          console.log('[Mock API] Student update - studentId:', studentId, 'data:', options?.json);

          const student = MockData.students.find(s => s.id === studentId);

          if (student && options?.json) {
            // Handle advisor_id assignment - also set advisor_name
            if (options.json.advisor_id) {
              student.advisor_id = options.json.advisor_id;
              const advisor = MockData.staff.find(s => s.id === options.json.advisor_id);
              if (advisor) {
                student.advisor = {
                  id: advisor.id,
                  first_name: advisor.first_name,
                  last_name: advisor.last_name,
                  email: advisor.email,
                };
                (student as any).advisor_name = `${advisor.first_name} ${advisor.last_name}`;
              }
              console.log('[Mock API] Advisor assigned:', student.advisor_id, '->', (student as any).advisor_name);
            }

            // Update other fields
            Object.assign(student, {
              ...options.json,
              updated_at: new Date().toISOString()
            });

            return {
              success: true,
              message: 'Student updated successfully',
              data: student
            };
          } else {
            console.warn('[Mock API] Student not found:', studentId);
          }
        }

        // Update staff profile
        if (cleanUrl === 'staff/profile' || (cleanUrl === 'profile' && this.prefixUrl?.includes('staff'))) {
          if (options?.json) {
            // Update mock staff profile
            Object.assign(MockData.staffProfile, options.json);

            // Also update the individual staff profile if ID is present in the body
            if (options.json.id && MockData.staffProfiles[options.json.id]) {
              Object.assign(MockData.staffProfiles[options.json.id], options.json);
            }

            return {
              success: true,
              message: 'Profile updated successfully',
              data: options.json
            };
          }
        }

        // Update admin staff profile
        if (cleanUrl === 'admin-staff/profile' || (cleanUrl === 'profile' && this.prefixUrl?.includes('admin-staff'))) {
          if (options?.json) {
            // Update individual admin staff profile if ID is present
            if (options.json.id && MockData.adminStaffProfiles[options.json.id]) {
              Object.assign(MockData.adminStaffProfiles[options.json.id], options.json);

              return {
                success: true,
                message: 'Admin profile updated successfully',
                data: MockData.adminStaffProfiles[options.json.id]
              };
            }
          }
        }

        return { success: true, message: 'Güncelleme başarılı' };
      }
    };
  }

  delete(url: string): MockResponse<any> {
    const fullUrl = this.prefixUrl ? `${this.prefixUrl}/${url}` : url;
    console.log('[Mock API] DELETE', fullUrl);

    return {
      json: async () => {
        await delay();

        const cleanUrl = fullUrl.replace(/^\//, '').replace(/^api\//, '');

        // Delete student
        if (cleanUrl.match(/^students\/[\w-]+$/)) {
          const studentId = cleanUrl.split('/').pop();
          const index = MockData.students.findIndex(s => s.id === studentId);

          if (index !== -1) {
            MockData.students.splice(index, 1);

            return {
              success: true,
              message: 'Student deleted successfully'
            };
          }
        }

        return { success: true, message: 'Silme başarılı' };
      }
    };
  }

  extend(options: { prefixUrl: string }) {
    const client = new MockApiClient();
    client.prefixUrl = options.prefixUrl;
    return client;
  }
}

// Create mock API client instances - URLs must match Traefik routing
const mockApiClient = new MockApiClient();

export const mockAuthApi = mockApiClient.extend({ prefixUrl: '/api/auth' });
export const mockStaffApi = mockApiClient.extend({ prefixUrl: '/api/staff' });
export const mockAdminStaffApi = mockApiClient.extend({ prefixUrl: '/api/admin-staff' });
export const mockStudentApi = mockApiClient.extend({ prefixUrl: '/api/students' });
export const mockCatalogApi = mockApiClient.extend({ prefixUrl: '/api/catalog' });
export const mockSemesterApi = mockApiClient.extend({ prefixUrl: '/api/semesters' });
export const mockEnrollmentApi = mockApiClient.extend({ prefixUrl: '/api/enrollment' });
export const mockAttendanceApi = mockApiClient.extend({ prefixUrl: '/api/attendance' });
export const mockGradesApi = mockApiClient.extend({ prefixUrl: '/api/grades' });
export const mockMealApi = mockApiClient.extend({ prefixUrl: '/api/meals' });
