'use client'

import React, { useEffect, useState } from "react";
import { getBaseUrl, saveAccessToken, saveRefreshToken } from "../util";
import { useRouter } from "next/navigation";

export default function CallbackPage() {
  const [data, setData] = useState(null)
  const router = useRouter()

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const state = searchParams.get("state");
    const code = searchParams.get("code");
    const url = getBaseUrl() + "/api/1/auth/callback?state=" + state + "&code=" + code;
    fetch(url, { method: 'GET' })
      .then(res => res.json())
        .then(json => {
          if (json.access_token) {
            saveAccessToken(json.access_token);
            saveRefreshToken(json.refresh_token);
            setData(json);
            router.push('/authorized');
          }
        })
        .catch(e => router.push('/loginfailed'))
      .catch(e => router.push('/loginfailed'));
  }, []);

  return (
    <main className="flex min-h-screen flex-col items-center justify-between p-24">
      <div className="relative flex place-items-center">
        Loading...
      </div>
    </main>
  )
}
