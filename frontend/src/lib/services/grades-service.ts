import { gradesApi } from '@/lib/api-client';
import type {
  CourseStatusResponse,
  CourseStudentsResponse,
  SubmitScoreRequest,
  SubmitScoreResponse,
  BulkSubmitScoresRequest,
  BulkSubmitScoresResponse,
} from '@/lib/types';

export const gradesService = {
  async getCourseStatus(courseId: string): Promise<CourseStatusResponse> {
    try {
      const response = await gradesApi.get(`course/${courseId}/status`).json<CourseStatusResponse>();
      return response;
    } catch (error) {
      console.error('Failed to fetch course status:', error);
      throw error;
    }
  },

  async getCourseStudents(courseId: string): Promise<CourseStudentsResponse> {
    try {
      const response = await gradesApi.get(`course/${courseId}/students`).json<CourseStudentsResponse>();
      return response;
    } catch (error) {
      console.error('Failed to fetch course students:', error);
      throw error;
    }
  },

  async submitScore(courseId: string, data: SubmitScoreRequest): Promise<SubmitScoreResponse> {
    try {
      const response = await gradesApi.post(`course/${courseId}/scores`, {
        json: data,
      }).json<SubmitScoreResponse>();
      return response;
    } catch (error) {
      console.error('Failed to submit score:', error);
      throw error;
    }
  },

  async bulkSubmitScores(courseId: string, data: BulkSubmitScoresRequest): Promise<BulkSubmitScoresResponse> {
    try {
      const response = await gradesApi.post(`course/${courseId}/scores/bulk`, {
        json: data,
      }).json<BulkSubmitScoresResponse>();
      return response;
    } catch (error) {
      console.error('Failed to bulk submit scores:', error);
      throw error;
    }
  },

  async unlockScore(data: { registration_id: string; slug: string }): Promise<void> {
    try {
      await gradesApi.post('admin/scores/unlock', {
        json: data,
      }).json();
    } catch (error) {
      console.error('Failed to unlock score:', error);
      throw error;
    }
  },

  async lockScore(data: { registration_id: string; slug: string }): Promise<void> {
    try {
      await gradesApi.post('admin/scores/lock', {
        json: data,
      }).json();
    } catch (error) {
      console.error('Failed to lock score:', error);
      throw error;
    }
  },
};

export default gradesService;
