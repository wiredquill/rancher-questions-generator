import React from 'react';
import {
  Box,
  Container,
  Typography,
  Grid,
  ThemeProvider,
  createTheme,
  CssBaseline,
} from '@mui/material';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { ChartInput } from './components/ChartInput';
import { ValuesExplorer } from './components/ValuesExplorer';
import { FormBuilder } from './components/FormBuilder';
import { useAppStore } from './store';

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#2196f3',
    },
  },
});

function App() {
  const { chartData } = useAppStore();

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <DndProvider backend={HTML5Backend}>
        <Container maxWidth="xl" sx={{ py: 3 }}>
          <Typography variant="h4" component="h1" gutterBottom>
            Rancher Questions Generator
          </Typography>
          <Typography variant="subtitle1" color="text.secondary" sx={{ mb: 3 }}>
            Generate questions.yaml files for Helm charts with drag-and-drop interface
          </Typography>

          <ChartInput />

          {chartData && (
            <Box sx={{ flexGrow: 1 }}>
              <Grid container spacing={3} sx={{ height: 'calc(100vh - 300px)' }}>
                <Grid item xs={12} md={6}>
                  <ValuesExplorer values={chartData.values} />
                </Grid>
                <Grid item xs={12} md={6}>
                  <FormBuilder />
                </Grid>
              </Grid>
            </Box>
          )}
        </Container>
      </DndProvider>
    </ThemeProvider>
  );
}

export default App;