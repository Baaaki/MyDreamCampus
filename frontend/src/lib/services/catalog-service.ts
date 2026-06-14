import { catalogApi } from '@/lib/api-client';
import type { CourseCatalog, CourseCoordinator, WeeklyTopic, Prerequisite } from '@/lib/types';

// API Response types
export interface CourseListItem {
  id: string;
  course_code: string;
  name: string;
  faculty: string;
  department: string;
  offering_unit?: string;
  class_level: number;
  semester?: number;
  credits: number;
  ects?: number;
  theoretical_hours: number;
  lab_hours: number;
  course_type: string;
  course_category: string;
  education_level: string;
  teaching_type: string;
  language: string;
  prerequisites: Prerequisite[];
  status: string;
}

export interface CourseResponse {
  id: string;
  course_code: string;
  name: string;
  faculty: string;
  department: string;
  offering_unit?: string;
  class_level: number;
  semester?: number;
  credits: number;
  ects?: number;
  theoretical_hours: number;
  lab_hours: number;
  course_type: string;
  course_category: string;
  education_level: string;
  teaching_type: string;
  language: string;
  prerequisites: Prerequisite[];
  coordinator?: CourseCoordinator;
  purpose?: string;
  description?: string;
  learning_outcomes?: string;
  learning_outcomes_list?: string[];
  weekly_topics?: WeeklyTopic[];
  recommended_sources?: string[];
  syllabus?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface PaginationResponse {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface ListCoursesResponse {
  data: CourseListItem[];
  pagination: PaginationResponse;
}

export interface ListCoursesParams {
  page?: number;
  limit?: number;
  faculty?: string;
  department?: string;
  course_type?: 'mandatory' | 'elective';
  course_category?: 'theoretical' | 'practical' | 'internship' | 'project' | 'seminar';
  education_level?: 'undergraduate' | 'graduate' | 'doctorate';
  status?: 'active' | 'draft' | 'pending_approval' | 'under_revision' | 'archived' | 'suspended';
  class_level?: number;
  semester?: number;
  language?: string;
  search?: string;
}

export interface CreateCourseRequest {
  course_code: string;
  name: string;
  faculty: string;
  department: string;
  offering_unit?: string;
  class_level: number;
  semester?: number;
  credits: number;
  ects?: number;
  theoretical_hours: number;
  lab_hours?: number;
  course_type: 'mandatory' | 'elective';
  course_category?: string;
  education_level?: string;
  teaching_type?: string;
  language?: string;
  prerequisites?: Prerequisite[];
  coordinator?: CourseCoordinator;
  purpose?: string;
  description?: string;
  learning_outcomes?: string;
  learning_outcomes_list?: string[];
  weekly_topics?: WeeklyTopic[];
  recommended_sources?: string[];
  syllabus?: string;
  status?: string;
}

export interface UpdateCourseRequest {
  name?: string;
  faculty?: string;
  department?: string;
  offering_unit?: string;
  class_level?: number;
  semester?: number;
  credits?: number;
  ects?: number;
  theoretical_hours?: number;
  lab_hours?: number;
  course_type?: 'mandatory' | 'elective';
  course_category?: string;
  education_level?: string;
  teaching_type?: string;
  language?: string;
  prerequisites?: Prerequisite[];
  coordinator?: CourseCoordinator;
  purpose?: string;
  description?: string;
  learning_outcomes?: string;
  learning_outcomes_list?: string[];
  weekly_topics?: WeeklyTopic[];
  recommended_sources?: string[];
  syllabus?: string;
  status?: string;
}

// Helper function to convert API response to frontend type
function mapCourseResponseToCourseCatalog(course: CourseResponse): CourseCatalog {
  return {
    id: course.id,
    course_code: course.course_code,
    name: course.name,
    faculty: course.faculty,
    department: course.department,
    offering_unit: course.offering_unit,
    class_level: course.class_level,
    semester: course.semester,
    credits: course.credits,
    ects: course.ects,
    theoretical_hours: course.theoretical_hours,
    lab_hours: course.lab_hours,
    course_type: course.course_type,
    education_level: mapEducationLevel(course.education_level),
    teaching_type: mapTeachingType(course.teaching_type),
    language: course.language,
    prerequisites: course.prerequisites || [],
    coordinator: course.coordinator,
    purpose: course.purpose,
    description: course.description,
    learning_outcomes: course.learning_outcomes,
    learning_outcomes_list: course.learning_outcomes_list,
    weekly_topics: course.weekly_topics,
    recommended_sources: course.recommended_sources,
    syllabus: course.syllabus,
    status: course.status,
    created_at: course.created_at,
    updated_at: course.updated_at,
  };
}

function mapCourseListItemToCourseCatalog(item: CourseListItem): CourseCatalog {
  return {
    id: item.id,
    course_code: item.course_code,
    name: item.name,
    faculty: item.faculty,
    department: item.department,
    offering_unit: item.offering_unit,
    class_level: item.class_level,
    semester: item.semester,
    credits: item.credits,
    ects: item.ects,
    theoretical_hours: item.theoretical_hours,
    lab_hours: item.lab_hours,
    course_type: item.course_type,
    education_level: mapEducationLevel(item.education_level),
    teaching_type: mapTeachingType(item.teaching_type),
    language: item.language,
    prerequisites: item.prerequisites || [],
    status: item.status,
    created_at: '',
    updated_at: '',
  };
}

// Map backend enum values to frontend display values
function mapEducationLevel(level: string): string {
  const mapping: Record<string, string> = {
    'undergraduate': 'Lisans',
    'graduate': 'Yüksek Lisans',
    'doctorate': 'Doktora',
  };
  return mapping[level] || level;
}

function mapTeachingType(type: string): string {
  const mapping: Record<string, string> = {
    'on_campus': 'Örgün Öğretim',
    'online': 'Uzaktan Öğretim',
    'hybrid': 'Hibrit',
  };
  return mapping[type] || type;
}

// Catalog Service
export const catalogService = {
  /**
   * List courses with optional filters
   */
  async listCourses(params?: ListCoursesParams): Promise<{ courses: CourseCatalog[]; pagination: PaginationResponse }> {
    const searchParams = new URLSearchParams();

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          searchParams.append(key, String(value));
        }
      });
    }

    const queryString = searchParams.toString();
    const url = queryString ? `courses?${queryString}` : 'courses';

    const response = await catalogApi.get(url).json<ListCoursesResponse>();

    return {
      courses: (response.data || []).map(mapCourseListItemToCourseCatalog),
      pagination: response.pagination,
    };
  },

  /**
   * Get all courses for a specific department
   */
  async getCoursesByDepartment(department: string): Promise<CourseCatalog[]> {
    const response = await catalogApi.get('courses', {
      searchParams: {
        department,
        limit: '100', // Backend max limit is 100
      },
    }).json<ListCoursesResponse>();

    return response.data.map(mapCourseListItemToCourseCatalog);
  },

  /**
   * Get a single course by course code
   */
  async getCourseByCode(courseCode: string): Promise<CourseCatalog> {
    const response = await catalogApi.get(`courses/${encodeURIComponent(courseCode)}`).json<CourseResponse>();
    return mapCourseResponseToCourseCatalog(response);
  },

  /**
   * Create a new course (Admin only)
   */
  async createCourse(data: CreateCourseRequest): Promise<CourseCatalog> {
    const response = await catalogApi.post('courses', { json: data }).json<CourseResponse>();
    return mapCourseResponseToCourseCatalog(response);
  },

  /**
   * Update an existing course (Admin only)
   */
  async updateCourse(courseCode: string, data: UpdateCourseRequest): Promise<CourseCatalog> {
    const response = await catalogApi.put(`courses/${encodeURIComponent(courseCode)}`, { json: data }).json<CourseResponse>();
    return mapCourseResponseToCourseCatalog(response);
  },
};

export default catalogService;
