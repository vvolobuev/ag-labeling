import { createRouter, createWebHistory } from 'vue-router'
import LandingView from '@/views/LandingView.vue'
import LoginView from '@/views/LoginView.vue'
import RegisterView from '@/views/RegisterView.vue'
import VerifyView from '@/views/VerifyView.vue'
import WorkspacesView from '@/views/WorkspacesView.vue'
import WorkspaceLayoutView from '@/views/WorkspaceLayoutView.vue'
import AppHomeView from '@/views/AppHomeView.vue'
import WorkspaceProjectsView from '@/views/WorkspaceProjectsView.vue'
import WorkspaceExploreView from '@/views/WorkspaceExploreView.vue'
import WorkspaceSettingsView from '@/views/WorkspaceSettingsView.vue'
import ProjectLayoutView from '@/views/ProjectLayoutView.vue'
import ProjectDataView from '@/views/ProjectDataView.vue'
import ProjectUploadView from '@/views/ProjectUploadView.vue'
import ProjectAnnotateHubView from '@/views/ProjectAnnotateHubView.vue'
import ProjectDatasetView from '@/views/ProjectDatasetView.vue'
import ProjectVersionsView from '@/views/ProjectVersionsView.vue'
import ProjectClassesTagsView from '@/views/ProjectClassesTagsView.vue'
import LegacyProjectRedirectView from '@/views/LegacyProjectRedirectView.vue'
import VersionView from '@/views/VersionView.vue'
import AnnotateView from '@/views/AnnotateView.vue'
import AdminView from '@/views/AdminView.vue'
import { getToken } from '@/composables/useApi'

const routes = [
  { path: '/', name: 'Landing', component: LandingView },
  { path: '/login', name: 'Login', component: LoginView, meta: { guest: true } },
  { path: '/register', name: 'Register', component: RegisterView, meta: { guest: true } },
  { path: '/verify', name: 'Verify', component: VerifyView },
  { path: '/workspaces', name: 'Workspaces', component: WorkspacesView, meta: { requiresAuth: true } },
  {
    path: '/app/:wid',
    component: WorkspaceLayoutView,
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: { name: 'AppProjects' } },
      { path: 'home', name: 'AppHome', component: AppHomeView, meta: { requiresAuth: true } },
      { path: 'projects', name: 'AppProjects', component: WorkspaceProjectsView, meta: { requiresAuth: true } },
      { path: 'explore', name: 'AppExplore', component: WorkspaceExploreView, meta: { requiresAuth: true } },
      { path: 'settings', name: 'AppSettings', component: WorkspaceSettingsView, meta: { requiresAuth: true } },
      {
        path: 'projects/:pid',
        component: ProjectLayoutView,
        meta: { requiresAuth: true },
        children: [
          { path: '', redirect: { name: 'AppProjectDataset' } },
          { path: 'data', name: 'AppProjectData', component: ProjectDataView, meta: { requiresAuth: true } },
          { path: 'upload', name: 'AppProjectUpload', component: ProjectUploadView, meta: { requiresAuth: true } },
          { path: 'annotate', name: 'AppProjectAnnotate', component: ProjectAnnotateHubView, meta: { requiresAuth: true } },
          { path: 'dataset', name: 'AppProjectDataset', component: ProjectDatasetView, meta: { requiresAuth: true } },
          { path: 'versions', name: 'AppProjectVersions', component: ProjectVersionsView, meta: { requiresAuth: true } },
          {
            path: 'classes-tags',
            name: 'AppProjectClassesTags',
            component: ProjectClassesTagsView,
            meta: { requiresAuth: true },
          },
        ],
      },
    ],
  },
  { path: '/workspaces/:wid', redirect: (to) => `/app/${to.params.wid}/projects` },
  { path: '/projects/:pid', name: 'ProjectLegacyRedirect', component: LegacyProjectRedirectView, meta: { requiresAuth: true } },
  { path: '/versions/:vid', name: 'Version', component: VersionView, meta: { requiresAuth: true } },
  { path: '/annotate/:imgid', name: 'Annotate', component: AnnotateView, meta: { requiresAuth: true } },
  { path: '/admin', name: 'Admin', component: AdminView },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const hasToken = !!getToken()
  if ((to.name === 'Landing' || to.meta.guest) && hasToken) {
    const last = localStorage.getItem('ag_last_workspace') || ''
    if (last) return `/app/${last}/projects`
    return { name: 'Workspaces' }
  }
  if (to.meta.requiresAuth && !getToken()) {
    return { name: 'Login', query: { redirect: to.fullPath } }
  }
})

export default router
