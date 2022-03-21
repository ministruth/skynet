import MainContainer from '@/common_components/layout/container';
import AgentCard from '@/components/config/agent';
import SettingCard from '@/components/config/setting';
import { PLUGIN_ID } from '@/config';
import { getIntl } from '@/utils';
import { Space } from 'antd';

const ConfigPage = () => {
  const intl = getIntl();
  return (
    <MainContainer
      title={intl.get('menus.monitor')}
      routes={[
        {
          path: '/plugin',
          breadcrumbName: 'menus.plugin',
        },
        {
          path: `/plugin/${PLUGIN_ID}/config`,
          breadcrumbName: 'menus.monitor',
        },
      ]}
      content={intl.get('pages.config.content')}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <SettingCard />
        <AgentCard />
      </Space>
    </MainContainer>
  );
};

ConfigPage.title = 'titles.monitor';

export default ConfigPage;
