import { enrollmentApi } from '@/lib/api-client';
import { AdvisorPendingProgramsResponse } from '@/lib/types';

export const enrollmentService = {
    /**
     * Get pending enrollment programs for the logged-in advisor
     */
    async getPendingEnrollments(): Promise<AdvisorPendingProgramsResponse> {
        try {
            // Use the correct endpoint path relative to enrollmentApi's base URL
            // enrollmentApi base is /api/enrollment, so this request goes to /api/enrollment/advisor/pending-programs
            const response = await enrollmentApi.get('advisor/pending-programs').json<AdvisorPendingProgramsResponse>();
            return response;
        } catch (error) {
            console.error('Failed to fetch pending enrollments:', error);
            throw error;
        }
    },

    /**
     * Approve an enrollment program
     */
    async approveEnrollment(programId: string): Promise<void> {
        try {
            await enrollmentApi.post(`advisor/programs/${programId}/approve`);
        } catch (error) {
            console.error(`Failed to approve enrollment ${programId}:`, error);
            throw error;
        }
    },

    /**
     * Reject an enrollment program
     */
    async rejectEnrollment(programId: string, rejectionReason: string): Promise<void> {
        try {
            await enrollmentApi.post(`advisor/programs/${programId}/reject`, {
                json: { rejection_reason: rejectionReason },
            });
        } catch (error) {
            console.error(`Failed to reject enrollment ${programId}:`, error);
            throw error;
        }
    },
};

export default enrollmentService;
