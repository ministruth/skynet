import { getAPI } from '@/utils';
import { useCallback, useState } from 'react';

export default () => {
  const [setting, setSetting] = useState<{ [Key: string]: any }>({});

  const getSetting = useCallback(async () => {
    const rsp = await getAPI('/setting/public');
    setSetting(rsp.data);
  }, []);

  return {
    setting,
    getSetting,
  };
};
