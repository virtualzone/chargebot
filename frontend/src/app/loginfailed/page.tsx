'use client'

import Link from "next/link"

export default function Authorized() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-between p-24">
      <div className="relative flex place-items-center">
        <p>Login failed.</p>
        <p><Link href="/">Back to start page</Link></p>
      </div>
    </main>
  )
}
