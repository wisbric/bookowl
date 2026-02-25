import { createContext, useContext, useState, type ReactNode } from 'react'

interface PanelContextValue {
  rightPanelOpen: boolean
  setRightPanelOpen: (open: boolean) => void
}

const PanelContext = createContext<PanelContextValue>({
  rightPanelOpen: false,
  setRightPanelOpen: () => {},
})

export function PanelProvider({ children }: { children: ReactNode }) {
  const [rightPanelOpen, setRightPanelOpen] = useState(false)
  return (
    <PanelContext.Provider value={{ rightPanelOpen, setRightPanelOpen }}>
      {children}
    </PanelContext.Provider>
  )
}

export function useRightPanel() {
  return useContext(PanelContext)
}
