import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import { ProjectsPage } from './pages/ProjectsPage'
import { SessionsPage } from './pages/SessionsPage'
import { SessionDetailPage } from './pages/SessionDetailPage'
import ProjectDetailPage from './pages/ProjectDetailPage'
import GroupDetailPage from './pages/GroupDetailPage'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<ProjectsPage />} />
          <Route path="projects" element={<ProjectsPage />} />
          <Route path="projects/:name" element={<ProjectDetailPage />} />
          <Route path="projects/:projectName/sessions" element={<SessionsPage />} />
          <Route path="projects/:projectName/sessions/:sessionId" element={<SessionDetailPage />} />
          <Route path="groups/:id" element={<GroupDetailPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
