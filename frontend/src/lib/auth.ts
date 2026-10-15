import { createContext, useContext } from 'react';
import type { Me } from './api';

export const MeContext = createContext<Me | null>(null);

/** Returns the signed-in user. Only use below the signed-in app shell. */
export function useMe(): Me {
  const me = useContext(MeContext);
  if (me === null) {
    throw new Error('useMe requires a signed-in user');
  }
  return me;
}

export function isAdmin(me: Me): boolean {
  return me.user.role === 'admin';
}

export function canUseStudio(me: Me): boolean {
  return isAdmin(me) || (me.memberships ?? []).some((item) => item.role === 'lead');
}

/** Admins and product leads can generate passports and manage product access. */
export function canManage(me: Me, productId: string): boolean {
  return (
    isAdmin(me) ||
    (me.memberships ?? []).some((item) => item.product_id === productId && item.role === 'lead')
  );
}

export const roleLabels: Record<string, string> = {
  admin: 'Admin',
  user: 'User',
  lead: 'Product lead',
  learner: 'Learner',
};
