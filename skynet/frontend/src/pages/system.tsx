import MainLayout from '@/components/layout';
import MainContainer from '@/components/layout/container';
import DangerZoneCard from '@/components/system/dangerzone';
import SettingCard from '@/components/system/setting';
import { getIntl, UserPerm } from '@/utils';

const System = () => {
  const intl = getIntl();
  return (
    <MainLayout
      title="titles.system"
      access="manage.system"
      perm={UserPerm.PermRead}
    >
      <MainContainer
        title={intl.get('menus.system')}
        routes={[
          {
            title: 'menus.system',
          },
        ]}
        content={intl.get('pages.system.content')}
      >
        <SettingCard style={{ marginBottom: '16px' }} />
        <DangerZoneCard />
      </MainContainer>
    </MainLayout>
  );
};

export default System;
