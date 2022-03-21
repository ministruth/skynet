import MainLayout from '@/components/layout';
import MainContainer from '@/components/layout/container';
import DangerZoneCard from '@/components/system/dangerzone';
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
            path: '/system',
            breadcrumbName: 'menus.system',
          },
        ]}
        content={intl.get('pages.system.content')}
      >
        <DangerZoneCard />
      </MainContainer>
    </MainLayout>
  );
};

export default System;
