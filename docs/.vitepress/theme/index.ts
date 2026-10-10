import { theme, useOpenapi } from 'vitepress-openapi/client'

import 'vitepress-openapi/dist/style.css'
import DefaultTheme from 'vitepress/theme'

import spec from '../../../openapi/external.openapi.json'

useOpenapi({ spec })

export default {
  extends: DefaultTheme,
  async enhanceApp({ app }) {
    theme.enhanceApp({ app })
  },
}
