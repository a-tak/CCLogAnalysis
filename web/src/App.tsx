import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import { ProjectsPage } from './pages/ProjectsPage'
import { SessionsPage } from './pages/SessionsPage'
import { SessionDetailPage } from './pages/SessionDetailPage'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<ProjectsPage />} />
          <Route path="projects/:projectName" element={<SessionsPage />} />
          <Route path="projects/:projectName/sessions/:sessionId" element={<SessionDetailPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
