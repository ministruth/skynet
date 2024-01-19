import { getAPI } from '@/utils';
import { useCallback, useState } from 'react';

export default () => {
  const [setting, setSetting] = useState<{ [Key: string]: any }>({});

  const getSetting = useCallback(async () => {
    const rsp = await getAPI('/settings/public');
    if (rsp != undefined) setSetting(rsp.data);
  }, []);

  return {
    setting,
    getSetting,
  };
};
