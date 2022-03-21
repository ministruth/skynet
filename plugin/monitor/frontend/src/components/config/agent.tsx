import { getIntl } from '@/utils';
import ProCard from '@ant-design/pro-card';

const AgentCard = () => {
  const intl = getIntl();
  return (
    <ProCard title={intl.get('pages.config.agent.title')} bordered></ProCard>
  );
};

export default AgentCard;
