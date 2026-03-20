import { useState, useEffect } from 'react';
import {
  Container,
  Title,
  Button,
  Table,
  Badge,
  ActionIcon,
  Group,
  Text,
  Loader,
  Center,
} from '@mantine/core';
import { IconPlus, IconEdit, IconTrash } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { modals } from '@mantine/modals';
import { problemsApi } from '../../api/problems';
import type { Problem } from '../../types/problem';
import { ProblemModal } from '../../components/admin/ProblemModal';

export function ProblemsPage() {
  const [problems, setProblems] = useState<Problem[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpened, setModalOpened] = useState(false);
  const [editingProblem, setEditingProblem] = useState<Problem | null>(null);

  const loadProblems = async () => {
    try {
      setLoading(true);
      const response = await problemsApi.list();
      setProblems(response.problems);
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: 'Failed to load problems',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProblems();
  }, []);

  const handleCreate = () => {
    setEditingProblem(null);
    setModalOpened(true);
  };

  const handleEdit = (problem: Problem) => {
    setEditingProblem(problem);
    setModalOpened(true);
  };

  const handleDelete = (problem: Problem) => {
    modals.openConfirmModal({
      title: 'Delete Problem',
      children: (
        <Text size="sm">
          Are you sure you want to delete "{problem.title}"? This action cannot be undone.
        </Text>
      ),
      labels: { confirm: 'Delete', cancel: 'Cancel' },
      confirmProps: { color: 'red' },
      onConfirm: async () => {
        try {
          await problemsApi.delete(problem.id);
          notifications.show({
            title: 'Success',
            message: 'Problem deleted successfully',
            color: 'green',
          });
          loadProblems();
        } catch (error) {
          notifications.show({
            title: 'Error',
            message: 'Failed to delete problem',
            color: 'red',
          });
        }
      },
    });
  };

  const handleModalClose = (shouldReload?: boolean) => {
    setModalOpened(false);
    setEditingProblem(null);
    if (shouldReload) {
      loadProblems();
    }
  };

  const getVisibilityColor = (visibility: string) => {
    switch (visibility) {
      case 'published':
        return 'green';
      case 'draft':
        return 'yellow';
      case 'archived':
        return 'gray';
      default:
        return 'blue';
    }
  };

  if (loading) {
    return (
      <Center h={400}>
        <Loader size="lg" />
      </Center>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <Title order={1}>Problems Management</Title>
        <Button leftSection={<IconPlus size={16} />} onClick={handleCreate}>
          Add Problem
        </Button>
      </Group>

      <Table striped highlightOnHover>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Slug</Table.Th>
            <Table.Th>Title</Table.Th>
            <Table.Th>Visibility</Table.Th>
            <Table.Th>Time Limit</Table.Th>
            <Table.Th>Memory Limit</Table.Th>
            <Table.Th>Actions</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {problems.length === 0 ? (
            <Table.Tr>
              <Table.Td colSpan={6}>
                <Text ta="center" c="dimmed">
                  No problems found
                </Text>
              </Table.Td>
            </Table.Tr>
          ) : (
            problems.map((problem) => (
              <Table.Tr key={problem.id}>
                <Table.Td>
                  <Text size="sm" fw={500}>
                    {problem.slug}
                  </Text>
                </Table.Td>
                <Table.Td>{problem.title}</Table.Td>
                <Table.Td>
                  <Badge color={getVisibilityColor(problem.visibility)}>
                    {problem.visibility}
                  </Badge>
                </Table.Td>
                <Table.Td>{problem.time_limit_ms}ms</Table.Td>
                <Table.Td>{problem.memory_limit_mb}MB</Table.Td>
                <Table.Td>
                  <Group gap="xs">
                    <ActionIcon
                      variant="subtle"
                      color="blue"
                      onClick={() => handleEdit(problem)}
                    >
                      <IconEdit size={16} />
                    </ActionIcon>
                    <ActionIcon
                      variant="subtle"
                      color="red"
                      onClick={() => handleDelete(problem)}
                    >
                      <IconTrash size={16} />
                    </ActionIcon>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))
          )}
        </Table.Tbody>
      </Table>

      <ProblemModal
        opened={modalOpened}
        onClose={handleModalClose}
        problem={editingProblem}
      />
    </Container>
  );
}
