import type { ReactNode } from 'react'
import styled from 'styled-components'

import { AppLayout } from './AppLayout'

const AdminPanel = styled.div`
  border: 1px dashed rgba(94, 234, 212, 0.28);
  border-radius: 18px;
  padding: 0.8rem;
  background: rgba(8, 20, 32, 0.5);
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
