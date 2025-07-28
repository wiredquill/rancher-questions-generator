import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  List,
  ListItem,
  ListItemText,
  IconButton,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  FormControlLabel,
  Checkbox,
  Chip,
} from '@mui/material';
import { Delete, Edit, Download } from '@mui/icons-material';
import { useDrop } from 'react-dnd';
import { Question, Questions } from '../types';
import { useAppStore } from '../store';
import { api } from '../services/api';

interface QuestionEditorProps {
  question: Question | null;
  open: boolean;
  onClose: () => void;
  onSave: (question: Question) => void;
}

const QuestionEditor: React.FC<QuestionEditorProps> = ({
  question,
  open,
  onClose,
  onSave,
}) => {
  const [formData, setFormData] = useState<Question>(() => ({
    variable: '',
    label: '',
    description: '',
    type: 'string',
    required: false,
    group: 'General',
    ...question,
  }));

  const handleSave = () => {
    onSave(formData);
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        {question ? 'Edit Question' : 'Create Question'}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
          <TextField
            label="Variable"
            value={formData.variable}
            onChange={(e) => setFormData({ ...formData, variable: e.target.value })}
            fullWidth
          />
          <TextField
            label="Label"
            value={formData.label}
            onChange={(e) => setFormData({ ...formData, label: e.target.value })}
            fullWidth
          />
          <TextField
            label="Description"
            value={formData.description || ''}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            multiline
            rows={2}
            fullWidth
          />
          <FormControl fullWidth>
            <InputLabel>Type</InputLabel>
            <Select
              value={formData.type || 'string'}
              label="Type"
              onChange={(e) => setFormData({ ...formData, type: e.target.value })}
            >
              <MenuItem value="string">String</MenuItem>
              <MenuItem value="int">Integer</MenuItem>
              <MenuItem value="boolean">Boolean</MenuItem>
              <MenuItem value="enum">Enum</MenuItem>
            </Select>
          </FormControl>
          <TextField
            label="Group"
            value={formData.group || ''}
            onChange={(e) => setFormData({ ...formData, group: e.target.value })}
            fullWidth
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={formData.required || false}
                onChange={(e) => setFormData({ ...formData, required: e.target.checked })}
              />
            }
            label="Required"
          />
          <TextField
            label="Default Value"
            value={formData.default || ''}
            onChange={(e) => setFormData({ ...formData, default: e.target.value })}
            fullWidth
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained">
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export const FormBuilder: React.FC = () => {
  const { chartData, updateQuestions } = useAppStore();
  const [editingQuestion, setEditingQuestion] = useState<Question | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);

  const questions = chartData?.questions.questions || [];

  const [{ isOver }, drop] = useDrop(() => ({
    accept: 'value-item',
    drop: (item: { path: string; value: any }) => {
      const newQuestion: Question = {
        variable: item.path,
        label: item.path.split('.').pop()?.replace(/([A-Z])/g, ' $1').replace(/^./, str => str.toUpperCase()) || item.path,
        type: typeof item.value === 'boolean' ? 'boolean' : typeof item.value === 'number' ? 'int' : 'string',
        group: 'General',
      };
      handleAddQuestion(newQuestion);
    },
    collect: (monitor) => ({
      isOver: monitor.isOver(),
    }),
  }));

  const handleAddQuestion = (question: Question) => {
    const updatedQuestions: Questions = {
      questions: [...questions, question],
    };
    updateQuestions(updatedQuestions);
  };

  const handleEditQuestion = (index: number) => {
    setEditingQuestion(questions[index]);
    setEditorOpen(true);
  };

  const handleSaveQuestion = (updatedQuestion: Question) => {
    const updatedQuestions: Questions = {
      questions: questions.map((q, i) => 
        editingQuestion && q.variable === editingQuestion.variable ? updatedQuestion : q
      ),
    };
    updateQuestions(updatedQuestions);
    setEditingQuestion(null);
  };

  const handleDeleteQuestion = (index: number) => {
    const updatedQuestions: Questions = {
      questions: questions.filter((_, i) => i !== index),
    };
    updateQuestions(updatedQuestions);
  };

  const handleDownload = async () => {
    if (!chartData) return;
    
    try {
      const yaml = await api.downloadQuestionsYaml(chartData.session_id);
      const blob = new Blob([yaml], { type: 'application/x-yaml' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'questions.yaml';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to download questions.yaml:', error);
    }
  };

  return (
    <>
      <Paper elevation={2} sx={{ p: 2, height: '100%' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">
            Form Builder
          </Typography>
          <Button
            startIcon={<Download />}
            onClick={handleDownload}
            disabled={!chartData}
          >
            Download YAML
          </Button>
        </Box>
        
        <Box
          ref={drop}
          sx={{
            minHeight: 200,
            border: '2px dashed',
            borderColor: isOver ? 'primary.main' : 'divider',
            borderRadius: 1,
            bgcolor: isOver ? 'action.hover' : 'transparent',
            p: 2,
          }}
        >
          {questions.length === 0 ? (
            <Typography variant="body2" color="text.secondary" textAlign="center">
              Drop values here to create questions
            </Typography>
          ) : (
            <List>
              {questions.map((question, index) => (
                <ListItem
                  key={question.variable}
                  secondaryAction={
                    <Box>
                      <IconButton onClick={() => handleEditQuestion(index)}>
                        <Edit />
                      </IconButton>
                      <IconButton onClick={() => handleDeleteQuestion(index)}>
                        <Delete />
                      </IconButton>
                    </Box>
                  }
                >
                  <ListItemText
                    primary={question.label}
                    secondary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 1 }}>
                        <Chip size="small" label={question.variable} variant="outlined" />
                        <Chip size="small" label={question.type || 'string'} color="primary" />
                        {question.required && <Chip size="small" label="required" color="error" />}
                        {question.group && <Chip size="small" label={question.group} />}
                      </Box>
                    }
                  />
                </ListItem>
              ))}
            </List>
          )}
        </Box>
      </Paper>

      <QuestionEditor
        question={editingQuestion}
        open={editorOpen}
        onClose={() => {
          setEditorOpen(false);
          setEditingQuestion(null);
        }}
        onSave={handleSaveQuestion}
      />
    </>
  );
};