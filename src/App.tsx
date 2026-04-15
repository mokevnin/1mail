import { trpc } from './utils/trpc.ts'

export default function App() {
  const hello = trpc.hello.useQuery({ name: 'TanStack' })

  if (!hello.data) return <div>Loading...</div>

  return (
    <div>
      <h1>1mail</h1>
      <p>{hello.data.greeting}</p>
    </div>
  )
}
