import { useState } from 'react';
import { Profile } from '../types';
import { fetchUserProfile, evolveUserProfile } from '../api';

export function useProfile(userID: string) {
  const [userProfile, setUserProfile] = useState<Profile>({
    user_id: '',
    preferences: [],
    facts: [],
  });
  const [isEvolving, setIsEvolving] = useState<boolean>(false);

  // 4. 获取用户长期记忆画像
  const loadProfile = async () => {
    if (!userID) return;
    try {
      const profile = await fetchUserProfile(userID);
      setUserProfile(profile);
    } catch (err) {
      console.error('加载画像失败:', err);
    }
  };

  // 5. 触发画像演化
  const evolveProfile = async (currentSessionID: string) => {
    if (!currentSessionID || isEvolving) return;
    setIsEvolving(true);
    try {
      const profile = await evolveUserProfile(userID, currentSessionID);
      setUserProfile(profile);
    } catch (err) {
      console.error('触发记忆演化失败:', err);
    } finally {
      setIsEvolving(false);
    }
  };

  return { userProfile, isEvolving, loadProfile, evolveProfile };
}
