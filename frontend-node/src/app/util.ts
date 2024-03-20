'use client'

export function getBaseUrl() {
  if (typeof window !== "undefined") {
    const currentLocation = window.location;
    return currentLocation.protocol + '//' + currentLocation.hostname + ":" + currentLocation.port;
  }
  return '';
}

export function deleteTokens() {
  if (typeof window !== "undefined") {
    window.localStorage.removeItem("accessToken");
    window.localStorage.removeItem("refreshToken");
    window.localStorage.removeItem("userDetails");
  }
}

export function saveAccessToken(token: string) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem("accessToken", token)
  }
}

export function saveRefreshToken(token: string) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem("refreshToken", token)
  }
}

export function saveUserDetails(details: any) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem("userDetails", JSON.stringify(details))
  }
}

export function getAccessToken() {
  if (typeof window !== "undefined") {
    return window.localStorage.getItem("accessToken")
  }
  return '';
}

export function getRefreshToken() {
  if (typeof window !== "undefined") {
    return window.localStorage.getItem("refreshToken")
  }
  return '';
}

export function getUserDetails() {
  if (typeof window !== "undefined") {
    let details = window.localStorage.getItem("userDetails")
    if ((details !== null) && (details !== undefined) && (details !== '')) {
      return JSON.parse(details)
    }
  }
  return {};
}

export async function getAPI(endpoint: string): Promise<any> {
  const res = await fetch(getBaseUrl() + endpoint, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${getAccessToken()}`,
    }
  });
  const json = await res.json();
  return json;
}

export async function deleteAPI(endpoint: string): Promise<any> {
  const res = await fetch(getBaseUrl() + endpoint, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${getAccessToken()}`,
    }
  });
  const json = await res.json();
  return json;
}

export async function postAPI(endpoint: string, data: any): Promise<any> {
  const res = await fetch(getBaseUrl() + endpoint, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${getAccessToken()}`,
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
      Authorization: `Bearer ${getAccessToken()}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  const json = await res.json();
  return json;
}

export async function checkAuth() {
  return new Promise<void>((resolve, reject) => {
    if (!getAccessToken()) {
      window.location.href = "/";
      reject();
      return;
    }

    fetch(getBaseUrl() + "/api/1/auth/tokenvalid", {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${getAccessToken()}`,
      }
    }).then(valid => {
      valid.json().then(validBool => {
        if (validBool) {
          resolve();
          return;
        }
        fetch(getBaseUrl() + "/api/1/auth/refresh", {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${getAccessToken()}`,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ access_token: getAccessToken(), refresh_token: getRefreshToken() }),
        }).then(resp => {
          if (resp.status !== 200) {
            deleteTokens();
            window.location.href = "/";
            reject();
            return;
          }
          resp.json().then(json => {
            saveAccessToken(json.access_token);
            saveRefreshToken(json.refresh_token);
            resolve();
          })
        });
      }).catch(() => {
        deleteTokens();
        window.location.href = "/";
        reject();
        return;
      });
    });
  });
}