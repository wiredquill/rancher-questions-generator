import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
} from '@mui/material';
import { ExpandMore } from '@mui/icons-material';
import { useDrag } from 'react-dnd';

interface ValueItemProps {
  path: string;
  value: any;
  level?: number;
}

const ValueItem: React.FC<ValueItemProps> = ({ path, value, level = 0 }) => {
  const [{ isDragging }, drag] = useDrag(() => ({
    type: 'value-item',
    item: { path, value },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  }));

  const isObject = typeof value === 'object' && value !== null && !Array.isArray(value);
  const isArray = Array.isArray(value);

  if (isObject) {
    return (
      <Accordion>
        <AccordionSummary expandIcon={<ExpandMore />}>
          <Typography variant="body2">
            {path.split('.').pop()} (object)
          </Typography>
        </AccordionSummary>
        <AccordionDetails>
          <Box sx={{ pl: 2 }}>
            {Object.entries(value).map(([key, val]) => (
              <ValueItem
                key={key}
                path={path ? `${path}.${key}` : key}
                value={val}
                level={level + 1}
              />
            ))}
          </Box>
        </AccordionDetails>
      </Accordion>
    );
  }

  return (
    <Box
      ref={drag}
      sx={{
        p: 1,
        mb: 1,
        bgcolor: isDragging ? 'action.selected' : 'background.default',
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: 1,
        cursor: 'grab',
        '&:hover': {
          bgcolor: 'action.hover',
        },
      }}
    >
      <Typography variant="body2" fontWeight="medium">
        {path}
      </Typography>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
        <Chip
          size="small"
          label={isArray ? 'array' : typeof value}
          color="primary"
          variant="outlined"
        />
        <Typography variant="caption" color="text.secondary">
          {isArray 
            ? `[${value.length} items]` 
            : String(value).length > 50 
            ? `${String(value).substring(0, 50)}...`
            : String(value)
          }
        </Typography>
      </Box>
    </Box>
  );
};

interface ValuesExplorerProps {
  values: Record<string, any>;
}

export const ValuesExplorer: React.FC<ValuesExplorerProps> = ({ values }) => {
  return (
    <Paper elevation={2} sx={{ p: 2, height: '100%' }}>
      <Typography variant="h6" gutterBottom>
        Values Explorer
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Drag values to the Form Builder to create questions
      </Typography>
      
      <Box sx={{ maxHeight: 'calc(100vh - 200px)', overflow: 'auto' }}>
        {Object.entries(values).map(([key, value]) => (
          <ValueItem key={key} path={key} value={value} />
        ))}
      </Box>
    </Paper>
  );
};