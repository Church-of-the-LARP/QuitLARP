export type AdminTab =
  | 'overview'
  | 'users'
  | 'assessments'
  | 'sessions'
  | 'submissions';

export type AdminAssessment = {
  id: number;
  title: string;
  description: string;
  difficulty: 'easy' | 'medium' | 'hard';
  timeLimitMinutes: number;
  author?: { username: string } | null;
};

export type AdminLiveSession = {
  id: string;
  userId: number;
  assessmentId: number;
  assessmentTitle: string;
  deadline: string;
  createdAt: string;
};

export type AdminSubmission = {
  id: number;
  assessmentId: number;
  assessmentTitle: string;
  userId: number;
  author?: { username: string } | null;
  public: boolean;
  commitSha: string;
  url: string;
  createdAt: string;
  updatedAt: string;
};
