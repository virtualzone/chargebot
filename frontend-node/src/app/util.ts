'use client'

export function getBaseUrl() {
  if (typeof window !== "undefined") {
    const currentLocation = window.location;
    return currentLocation.protocol + '//' + currentLocation.hostname + ":" + currentLocation.port;
  }
  return '';
}

export async function getAPI(endpoint: string): Promise<any> {
  const res = await fetch(getBaseUrl() + endpoint, {
    method: 'GET',
    headers: {
    }
  });
  const json = await res.json();
  return json;
}

export async function deleteAPI(endpoint: string): Promise<any> {
  const res = await fetch(getBaseUrl() + endpoint, {
    method: 'DELETE',
    headers: {
    }
  });
  const json = await res.json();
  return json;
}

export async function postAPI(endpoint: string, data: any): Promise<any> {
  const res = await fetch(getBaseUrl() + endpoint, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  const json = await res.json();
  return json;
}

export async function putAPI(endpoint: string, data: any): Promise<any> {
  const res = await fetch(getBaseUrl() + endpoint, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  const json = await res.json();
  return json;
}
