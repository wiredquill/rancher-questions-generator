export interface Question {
  variable: string;
  label: string;
  description?: string;
  type?: string;
  required?: boolean;
  default?: any;
  group?: string;
  options?: string[];
  show_if?: string;
  subquestions?: Question[];
}

export interface Questions {
  questions: Question[];
}

export interface ChartData {
  session_id: string;
  values: Record<string, any>;
  questions: Questions;
}

export interface AppState {
  chartData: ChartData | null;
  loading: boolean;
  error: string | null;
  setChartData: (data: ChartData) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  processChart: (url: string) => Promise<void>;
  updateQuestions: (questions: Questions) => Promise<void>;
}