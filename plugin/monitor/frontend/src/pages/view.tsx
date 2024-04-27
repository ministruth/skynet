import MainLayout from '@/common_components/layout';
import MainContainer from '@/common_components/layout/container';
import ViewCard from '@/components/view/card';
import { PLUGIN_ID } from '@/config';
import { UserPerm, getIntl } from '@/utils';

const ConfigPage = () => {
  const intl = getIntl();
  return (
    <MainLayout
      title="titles.monitor"
      access={`view.plugin.${PLUGIN_ID}`}
      perm={UserPerm.PermRead}
    >
      <MainContainer
        title={intl.get('menus.monitor')}
        routes={[
          {
            title: 'menus.service',
          },
          {
            title: 'menus.monitor',
          },
        ]}
        content={intl.get('pages.view.content')}
      >
        <ViewCard />
      </MainContainer>
    </MainLayout>
  );
};

ConfigPage.title = 'titles.monitor';

export default ConfigPage;
