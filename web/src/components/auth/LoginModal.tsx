import { Modal, TextInput, PasswordInput, Button, Stack } from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';
import type { LoginRequest } from '../../types/auth';

interface LoginModalProps {
  opened: boolean;
  onClose: () => void;
}

export function LoginModal({ opened, onClose }: LoginModalProps) {
  const { login } = useAuth();

  const form = useForm<LoginRequest>({
    initialValues: {
      email: '',
      password: '',
    },
    validate: {
      email: (value) => (!value ? 'Email is required' : !/^\S+@\S+$/.test(value) ? 'Invalid email' : null),
      password: (value) => (!value ? 'Password is required' : null),
    },
  });

  const handleSubmit = async (values: LoginRequest) => {
    try {
      await login(values);
      notifications.show({
        title: 'Success',
        message: 'Logged in successfully!',
        color: 'teal',
      });
      onClose();
      form.reset();
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: error instanceof Error ? error.message : 'Failed to login',
        color: 'red',
      });
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title="Login" centered>
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack gap="md">
          <TextInput
            label="Email"
            placeholder="your@email.com"
            type="email"
            required
            {...form.getInputProps('email')}
          />
          <PasswordInput
            label="Password"
            placeholder="••••••••"
            required
            {...form.getInputProps('password')}
          />
          <Button type="submit" fullWidth>
            Login
          </Button>
        </Stack>
      </form>
    </Modal>
  );
}
