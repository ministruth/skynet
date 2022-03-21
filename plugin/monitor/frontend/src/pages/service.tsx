import MainContainer from '@/common_components/layout/container';
import MonitorCard from '@/components/config/monitor';
import { PLUGIN_ID } from '@/config';
import { getIntl } from '@/utils';

const ServicePage = () => {
  const intl = getIntl();
  return (
    <MainContainer
      title={intl.get('menus.monitor')}
      routes={[
        {
          path: '/',
          breadcrumbName: 'menus.service',
        },
        {
          path: `/plugin/${PLUGIN_ID}/service`,
          breadcrumbName: 'menus.monitor',
        },
      ]}
      content={intl.get('pages.service.content')}
    >
      <MonitorCard />
    </MainContainer>
  );
};

ServicePage.title = 'titles.monitor';

export default ServicePage;
