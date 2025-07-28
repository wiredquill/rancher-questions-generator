import { create } from 'zustand';
import { AppState, ChartData, Questions } from '../types';
import { api } from '../services/api';

export const useAppStore = create<AppState>((set, get) => ({
  chartData: null,
  loading: false,
  error: null,

  setChartData: (chartData: ChartData) => set({ chartData }),
  setLoading: (loading: boolean) => set({ loading }),
  setError: (error: string | null) => set({ error }),

  processChart: async (url: string) => {
    set({ loading: true, error: null });
    try {
      const data = await api.processChart(url);
      set({ chartData: data, loading: false });
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Unknown error', 
        loading: false 
      });
    }
  },

  updateQuestions: async (questions: Questions) => {
    const { chartData } = get();
    if (!chartData) return;

    set({ loading: true, error: null });
    try {
      await api.updateQuestions(chartData.session_id, questions);
      set({ 
        chartData: { ...chartData, questions },
        loading: false 
      });
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Unknown error', 
        loading: false 
      });
    }
  },
}));