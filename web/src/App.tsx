import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Container, Title, Text } from '@mantine/core';
import { Navbar } from './components/layout/Navbar';
import { ProtectedRoute } from './components/ProtectedRoute';
import { ProblemsPage } from './pages/admin/ProblemsPage';

function HomePage() {
  return (
    <Container size="xl" className="py-8">
      <Title order={2} className="mb-4">
        Welcome to Capstone
      </Title>
      <Text c="dimmed">
        Your application content goes here...
      </Text>
    </Container>
  );
}

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-gray-50">
        <Navbar />
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route
            path="/admin/problems"
            element={
              <ProtectedRoute requiredPermission="admin.access">
                <ProblemsPage />
              </ProtectedRoute>
            }
          />
        </Routes>
      </div>
    </BrowserRouter>
  );
}

export default App;
