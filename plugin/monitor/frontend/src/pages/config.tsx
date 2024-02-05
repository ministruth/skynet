import MainLayout from '@/common_components/layout';
import MainContainer from '@/common_components/layout/container';
import AgentCard from '@/components/config/agent';
import SettingCard from '@/components/config/setting';
import { UserPerm, getIntl } from '@/utils';
import { Space } from 'antd';

const ConfigPage = () => {
  const intl = getIntl();
  return (
    <MainLayout
      title="titles.monitor"
      access="manage.plugin"
      perm={UserPerm.PermRead}
    >
      <MainContainer
        title={intl.get('menus.monitor')}
        routes={[
          {
            title: 'menus.plugin',
          },
          {
            title: 'menus.monitor',
          },
        ]}
        content={intl.get('pages.config.content')}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <SettingCard />
          <AgentCard />
        </Space>
      </MainContainer>
    </MainLayout>
  );
};

ConfigPage.title = 'titles.monitor';

export default ConfigPage;
