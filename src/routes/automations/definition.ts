import type { WorkflowBuilderEdge, WorkflowBuilderNode } from '@workflowbuilder/sdk'
import { EMAIL_NODE_ICON, EMAIL_NODE_TYPE, WAIT_NODE_ICON, WAIT_NODE_TYPE } from './nodes.tsx'

// AutomationStep mirrors the engine's step JSON ({type, subject, body, seconds}).
// It is the on-disk format the backend executor reads — keep it byte-compatible.
export interface AutomationStep {
  type: 'email' | 'wait'
  subject: string
  body: string
  seconds: number
}

export function emptyWaitStep(): AutomationStep {
  return { type: 'wait', subject: '', body: '', seconds: 3600 }
}

// serializeSteps turns steps into the JSON definition the engine reads, dropping
// fields irrelevant to each step type.
export function serializeSteps(steps: AutomationStep[]): string {
  return JSON.stringify(
    steps.map((s) =>
      s.type === 'email'
        ? { type: 'email', subject: s.subject, body: s.body }
        : { type: 'wait', seconds: Math.trunc(Number(s.seconds)) || 0 },
    ),
  )
}

// parseSteps reads a stored definition back into steps, tolerating an empty or
// malformed value (new automations default to "[]").
export function parseSteps(definition: string): AutomationStep[] {
  try {
    const raw = JSON.parse(definition || '[]')
    if (!Array.isArray(raw)) return []
    return raw.map((s: Partial<AutomationStep>) => ({
      type: s.type === 'wait' ? 'wait' : 'email',
      subject: s.subject ?? '',
      body: s.body ?? '',
      seconds: s.seconds ?? 3600,
    }))
  } catch {
    return []
  }
}

// --- graph ↔ steps (the visual builder uses an xyflow graph; we persist []step) ---

// The SDK's default node renderer key (NodeType.Node) and default edge type.
const NODE_RENDERER_TYPE = 'node'
const EDGE_TYPE = 'labelEdge'
// Default node handle ids the SDK's node template uses for a single in/out port.
const SOURCE_HANDLE = 'source'
const TARGET_HANDLE = 'target'
// Vertical spacing for the synthesized chain (layout is not persisted — we lay
// the chain out top-to-bottom on load and let the user rearrange freely).
const Y_GAP = 160

export interface AutomationGraph {
  nodes: WorkflowBuilderNode[]
  edges: WorkflowBuilderEdge[]
}

// stepsToGraph builds a linear top-to-bottom chain of nodes for the canvas from
// the stored steps. Positions are synthesized from order, never persisted.
export function stepsToGraph(steps: AutomationStep[]): AutomationGraph {
  const nodes: WorkflowBuilderNode[] = steps.map((step, i) => ({
    id: `step-${i}`,
    type: NODE_RENDERER_TYPE,
    position: { x: 0, y: i * Y_GAP },
    data: {
      type: step.type === 'wait' ? WAIT_NODE_TYPE : EMAIL_NODE_TYPE,
      icon: step.type === 'wait' ? WAIT_NODE_ICON : EMAIL_NODE_ICON,
      segments: [],
      properties:
        step.type === 'wait'
          ? { seconds: step.seconds }
          : { subject: step.subject, body: step.body },
    },
  }))

  const edges: WorkflowBuilderEdge[] = []
  for (let i = 1; i < steps.length; i++) {
    edges.push({
      id: `edge-${i - 1}-${i}`,
      source: `step-${i - 1}`,
      target: `step-${i}`,
      sourceHandle: SOURCE_HANDLE,
      targetHandle: TARGET_HANDLE,
      type: EDGE_TYPE,
    })
  }

  return { nodes, edges }
}

function nodeToStep(node: WorkflowBuilderNode): AutomationStep | null {
  const props = (node.data.properties ?? {}) as Record<string, unknown>
  if (node.data.type === WAIT_NODE_TYPE) {
    return { ...emptyWaitStep(), seconds: Math.trunc(Number(props.seconds)) || 0 }
  }
  if (node.data.type === EMAIL_NODE_TYPE) {
    return {
      type: 'email',
      subject: String(props.subject ?? ''),
      body: String(props.body ?? ''),
      seconds: 0,
    }
  }
  return null
}

// graphToSteps linearizes the canvas graph into ordered steps by walking from
// the entry node (in-degree 0) along single outgoing edges. Branches are not
// supported yet: extra nodes off the main chain are reported via `dropped` so
// the caller can warn instead of silently losing them.
export function graphToSteps(graph: AutomationGraph): { steps: AutomationStep[]; dropped: number } {
  const { nodes, edges } = graph
  if (nodes.length === 0) return { steps: [], dropped: 0 }

  const byId = new Map(nodes.map((n) => [n.id, n]))
  const outgoing = new Map<string, string[]>()
  const indegree = new Map<string, number>(nodes.map((n) => [n.id, 0]))
  for (const e of edges) {
    if (!byId.has(e.source) || !byId.has(e.target)) continue
    outgoing.set(e.source, [...(outgoing.get(e.source) ?? []), e.target])
    indegree.set(e.target, (indegree.get(e.target) ?? 0) + 1)
  }

  const root = nodes.find((n) => (indegree.get(n.id) ?? 0) === 0) ?? nodes[0]
  if (!root) return { steps: [], dropped: 0 }
  const steps: AutomationStep[] = []
  const visited = new Set<string>()
  let current: string | undefined = root.id
  while (current && !visited.has(current)) {
    visited.add(current)
    const node = byId.get(current)
    const step = node ? nodeToStep(node) : null
    if (step) steps.push(step)
    current = outgoing.get(current)?.[0]
  }

  return { steps, dropped: nodes.length - visited.size }
}
