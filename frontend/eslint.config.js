import { defineConfig, globalIgnores } from 'eslint/config'
import globals from 'globals'
import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'

export default defineConfig([
  {
    name: 'app/files-to-lint',
    files: ['**/*.{js,mjs,jsx,vue}'],
  },

  globalIgnores([
    '**/dist/**',
    '**/dist-ssr/**',
    '**/coverage/**',
    '**/src/views/MainPage.vue',
    '**/src/views/DepartmentPage.vue',
    '**/src/views/Appointment.vue',
    '**/src/views/HeaderPanel.vue',
    '**/src/views/AboutSection.vue',
    '**/src/views/Vacancies.vue',
    '**/src/components/FooterPanel.vue',
    '**/src/components/DoctorsCarousel.vue',
    '**/src/components/BannerFullWidth.vue',
    '**/src/components/BannerFullWidthDesktop.vue',
    '**/src/components/DepartmentsList.vue',
    '**/src/components/DoctorsList.vue',
    '**/src/components/LocationMap.vue',
    '**/src/components/Tabs.vue',
    '**/src/components/DoctorCard.vue',
    '**/src/components/NavigationMenu/**',
  ]),

  {
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
  },

  js.configs.recommended,
  ...pluginVue.configs['flat/essential'],
  skipFormatting,
])
