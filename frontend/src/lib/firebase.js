import { initializeApp } from 'firebase/app'
import { getFirestore } from 'firebase/firestore'

const app = initializeApp({
  apiKey: 'AIzaSyCBGWJwA7mPdezYCaZxO_LwhY_Uhj7IMRc',
  authDomain: 'micro-trading-cb6f5.firebaseapp.com',
  projectId: 'micro-trading-cb6f5',
  storageBucket: 'micro-trading-cb6f5.firebasestorage.app',
  messagingSenderId: '627586413250',
  appId: '1:627586413250:web:81311eda5399ad2dbac15b',
})

export const db = getFirestore(app)

export function fmtTs(ts) {
  if (!ts) return '—'
  const d = ts.toDate ? ts.toDate() : new Date(ts)
  return d.toLocaleString('sv-SE', { timeZone: 'Asia/Seoul' }).replace('T', ' ')
}
