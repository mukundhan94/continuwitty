import {
  Children,
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from 'react'
import styled, { css } from 'styled-components'

const Wrapper = styled.div<{ $visible: boolean }>`
  ${({ $visible }) =>
    !$visible &&
    css`
      opacity: 0;
    `}

  ${({ $visible }) =>
    $visible &&
    css`
      opacity: 1;
      transition: opacity 400ms ease;
    `}

  @media (prefers-reduced-motion: reduce) {
    opacity: 1;
  }
`

interface AnimatedSvgDiagramProps {
  children: ReactNode
  className?: string
  threshold?: number
}

export function AnimatedSvgDiagram({
  children,
  className,
  threshold = 0.2,
}: AnimatedSvgDiagramProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(() => {
    if (typeof window === 'undefined') {
      return false
    }
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  })

  useEffect(() => {
    if (visible) {
      return
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisible(true)
          observer.disconnect()
        }
      },
      { threshold },
    )

    if (ref.current) observer.observe(ref.current)
    return () => observer.disconnect()
  }, [threshold, visible])

  // Forward the 'visible' class to the direct child so that &.visible CSS
  // selectors defined on diagram wrapper styled-components actually fire.
  // (styled-components v6 requires the class to live on the element itself,
  //  not just on an ancestor.)
  const child = Children.only(children)
  if (!isValidElement<{ className?: string }>(child)) {
    return (
      <Wrapper ref={ref} $visible={visible} className={className}>
        {children}
      </Wrapper>
    )
  }

  const typedChild = child as ReactElement<{ className?: string }>
  const childCls = [typedChild.props.className, visible ? 'visible' : '']
    .filter(Boolean)
    .join(' ')

  return (
    <Wrapper ref={ref} $visible={visible} className={className}>
      {cloneElement(typedChild, { className: childCls })}
    </Wrapper>
  )
}
