import { useEffect } from 'react';
import {
  Modal,
  TextInput,
  NumberInput,
  Select,
  Button,
  Stack,
  Group,
  Text,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { RichTextEditor, Link } from '@mantine/tiptap';
import { useEditor } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import Underline from '@tiptap/extension-underline';
import { notifications } from '@mantine/notifications';
import { problemsApi } from '../../api/problems';
import type { Problem, ProblemVisibility } from '../../types/problem';

interface ProblemModalProps {
  opened: boolean;
  onClose: (shouldReload?: boolean) => void;
  problem: Problem | null;
}

export function ProblemModal({ opened, onClose, problem }: ProblemModalProps) {
  const isEditing = !!problem;

  const form = useForm({
    initialValues: {
      slug: '',
      title: '',
      time_limit_ms: 1000,
      memory_limit_mb: 256,
      tests_ref: '',
      visibility: 'draft' as ProblemVisibility,
    },
    validate: {
      title: (value) => (!value ? 'Title is required' : null),
      time_limit_ms: (value) => (value <= 0 ? 'Must be positive' : null),
      memory_limit_mb: (value) => (value <= 0 ? 'Must be positive' : null),
      tests_ref: (value) => (!value ? 'Tests reference is required' : null),
    },
  });

  const editor = useEditor({
    extensions: [
      StarterKit,
      Underline,
      Link,
    ],
    content: '',
  });

  useEffect(() => {
    if (!editor) return;

    if (problem && opened) {
      form.setValues({
        slug: problem.slug,
        title: problem.title,
        time_limit_ms: problem.time_limit_ms,
        memory_limit_mb: problem.memory_limit_mb,
        tests_ref: problem.tests_ref,
        visibility: problem.visibility,
      });
      editor.commands.setContent(problem.statement_markdown);
    } else if (!opened) {
      form.reset();
      editor.commands.setContent('');
    }
  }, [problem, opened, editor]);

  const handleSubmit = async (values: typeof form.values) => {
    const statementMarkdown = editor?.getHTML() || '';
    
    if (!statementMarkdown) {
      notifications.show({
        title: 'Error',
        message: 'Problem statement is required',
        color: 'red',
      });
      return;
    }

    try {
      const data = {
        ...values,
        statement_markdown: statementMarkdown,
      };

      if (isEditing) {
        await problemsApi.update(problem.id, data);
        notifications.show({
          title: 'Success',
          message: 'Problem updated successfully',
          color: 'green',
        });
      } else {
        await problemsApi.create(data);
        notifications.show({
          title: 'Success',
          message: 'Problem created successfully',
          color: 'green',
        });
      }

      onClose(true);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || `Failed to ${isEditing ? 'update' : 'create'} problem`,
        color: 'red',
      });
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={() => onClose()}
      title={isEditing ? 'Edit Problem' : 'Create Problem'}
      size="xl"
    >
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack gap="md">
          <TextInput
            label="Title"
            placeholder="Two Sum"
            required
            {...form.getInputProps('title')}
          />

          {isEditing && (
            <TextInput
              label="Slug"
              placeholder="two-sum"
              disabled
              {...form.getInputProps('slug')}
              description="Slug is auto-generated and cannot be changed"
            />
          )}

          <div>
            <Text size="sm" fw={500} mb={4}>
              Problem Statement <span style={{ color: 'red' }}>*</span>
            </Text>
            <RichTextEditor editor={editor}>
              <RichTextEditor.Toolbar sticky stickyOffset={60}>
                <RichTextEditor.ControlsGroup>
                  <RichTextEditor.Bold />
                  <RichTextEditor.Italic />
                  <RichTextEditor.Underline />
                  <RichTextEditor.Strikethrough />
                  <RichTextEditor.ClearFormatting />
                  <RichTextEditor.Code />
                </RichTextEditor.ControlsGroup>

                <RichTextEditor.ControlsGroup>
                  <RichTextEditor.H1 />
                  <RichTextEditor.H2 />
                  <RichTextEditor.H3 />
                  <RichTextEditor.H4 />
                </RichTextEditor.ControlsGroup>

                <RichTextEditor.ControlsGroup>
                  <RichTextEditor.Blockquote />
                  <RichTextEditor.Hr />
                  <RichTextEditor.BulletList />
                  <RichTextEditor.OrderedList />
                </RichTextEditor.ControlsGroup>

                <RichTextEditor.ControlsGroup>
                  <RichTextEditor.Link />
                  <RichTextEditor.Unlink />
                </RichTextEditor.ControlsGroup>

                <RichTextEditor.ControlsGroup>
                  <RichTextEditor.Undo />
                  <RichTextEditor.Redo />
                </RichTextEditor.ControlsGroup>
              </RichTextEditor.Toolbar>

              <RichTextEditor.Content style={{ minHeight: 200 }} />
            </RichTextEditor>
          </div>

          <Group grow>
            <NumberInput
              label="Time Limit (ms)"
              placeholder="1000"
              min={1}
              required
              {...form.getInputProps('time_limit_ms')}
            />

            <NumberInput
              label="Memory Limit (MB)"
              placeholder="256"
              min={1}
              required
              {...form.getInputProps('memory_limit_mb')}
            />
          </Group>

          <TextInput
            label="Tests Reference"
            placeholder="s3://bucket/tests/two-sum"
            required
            {...form.getInputProps('tests_ref')}
          />

          <Select
            label="Visibility"
            data={[
              { value: 'draft', label: 'Draft' },
              { value: 'published', label: 'Published' },
              { value: 'archived', label: 'Archived' },
            ]}
            required
            {...form.getInputProps('visibility')}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={() => onClose()}>
              Cancel
            </Button>
            <Button type="submit">{isEditing ? 'Update' : 'Create'}</Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}
