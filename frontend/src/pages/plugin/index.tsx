import MainLayout from '@/components/layout';
import MainContainer from '@/components/layout/container';
import PluginCard from '@/components/plugin/card';
import { getIntl, UserPerm } from '@/utils';

const Plugin = () => {
  const intl = getIntl();
  return (
    <MainLayout
      title="titles.plugin"
      access="manage.plugin"
      perm={UserPerm.PermRead}
    >
      <MainContainer
        title={intl.get('menus.plugin')}
        routes={[
          {
            path: '/plugin',
            breadcrumbName: 'menus.plugin',
          },
        ]}
        content={intl.get('pages.plugin.content')}
      >
        <PluginCard />
      </MainContainer>
    </MainLayout>
  );
};

export default Plugin;
