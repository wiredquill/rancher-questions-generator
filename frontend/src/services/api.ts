import { ChartData, Questions } from '../types';

const API_BASE = '/api';

export const api = {
  async processChart(url: string): Promise<ChartData> {
    const response = await fetch(`${API_BASE}/chart`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ url }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to process chart');
    }

    return response.json();
  },

  async getChart(sessionId: string): Promise<ChartData> {
    const response = await fetch(`${API_BASE}/chart/${sessionId}`);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to get chart');
    }

    return response.json();
  },

  async updateQuestions(sessionId: string, questions: Questions): Promise<void> {
    const response = await fetch(`${API_BASE}/chart/${sessionId}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(questions),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to update questions');
    }
  },

  async downloadQuestionsYaml(sessionId: string): Promise<string> {
    const response = await fetch(`${API_BASE}/chart/${sessionId}/q`);

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to download questions.yaml');
    }

    return response.text();
  },
};