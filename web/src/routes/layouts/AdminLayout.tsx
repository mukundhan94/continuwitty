import type { ReactNode } from 'react'
import styled from 'styled-components'

import { AppLayout } from './AppLayout'

const AdminPanel = styled.div`
  border: 1px dashed var(--color-line);
  border-radius: 18px;
  padding: 0.8rem;
  background: var(--surface-raised);
  min-height: 0;
`

interface AdminLayoutProps {
  title: string
  description: string
  children: ReactNode
}

export function AdminLayout({ title, description, children }: AdminLayoutProps) {
  return (
    <AppLayout title={title} description={description}>
      <AdminPanel>{children}</AdminPanel>
    </AppLayout>
  )
}
