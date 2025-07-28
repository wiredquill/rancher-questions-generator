import React, { useState } from 'react';
import {
  Box,
  TextField,
  Button,
  Typography,
  Paper,
  Alert,
} from '@mui/material';
import { useAppStore } from '../store';

export const ChartInput: React.FC = () => {
  const [url, setUrl] = useState('');
  const { processChart, loading, error } = useAppStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (url.trim()) {
      await processChart(url.trim());
    }
  };

  return (
    <Paper elevation={2} sx={{ p: 3, mb: 3 }}>
      <Typography variant="h6" gutterBottom>
        Chart URL
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Enter a URL to a Helm chart (.tgz) or OCI registry (oci://)
      </Typography>
      
      <Box component="form" onSubmit={handleSubmit} sx={{ display: 'flex', gap: 2 }}>
        <TextField
          fullWidth
          label="Chart URL"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://example.com/chart.tgz or oci://registry.io/chart:version"
          disabled={loading}
        />
        <Button
          type="submit"
          variant="contained"
          disabled={loading || !url.trim()}
          sx={{ minWidth: 120 }}
        >
          {loading ? 'Processing...' : 'Load Chart'}
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mt: 2 }}>
          {error}
        </Alert>
      )}
    </Paper>
  );
};