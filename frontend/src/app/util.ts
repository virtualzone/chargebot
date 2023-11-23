'use client'

export function getBaseUrl() {
  if (typeof window !== "undefined") {
    const currentLocation = window.location;
    return currentLocation.protocol + '//' + currentLocation.hostname + ":" + currentLocation.port;
  }
  return '';
}

export function saveAccessToken(token: string) {
  if (typeof window !== "undefined") {
    window.sessionStorage.setItem("accessToken", token)
  }
}

export function saveRefreshToken(token: string) {
  if (typeof window !== "undefined") {
    window.sessionStorage.setItem("refreshToken", token)
  }
}

export function getAccessToken() {
  if (typeof window !== "undefined") {
    return window.sessionStorage.getItem("accessToken")
  }
  return '';
}

export function getRefreshToken() {
  if (typeof window !== "undefined") {
    return window.sessionStorage.getItem("refreshToken")
  }
  return '';
}