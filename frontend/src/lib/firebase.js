import { initializeApp } from 'firebase/app'
import { getFirestore } from 'firebase/firestore'

const app = initializeApp({
  apiKey: 'AIzaSyCkC3NzuK5fiHvnEL-1sXS7c-MgPX4MgpE',
  authDomain: 'micro-trading-495614.firebaseapp.com',
  projectId: 'micro-trading-495614',
  storageBucket: 'micro-trading-495614.firebasestorage.app',
  messagingSenderId: '587135484750',
  appId: '1:587135484750:web:4443e0c6e5124bf261141f',
})

export const db = getFirestore(app)

export function fmtTs(ts) {
  if (!ts) return '—'
  const d = ts.toDate ? ts.toDate() : new Date(ts)
  return d.toLocaleString('sv-SE', { timeZone: 'Asia/Seoul' }).replace('T', ' ')
}
