import { getIntl } from '@/utils';
import { Helmet, MicroAppLink } from '@umijs/max';
import { Button, Result } from 'antd';
import { FormattedMessage } from 'react-intl';

const DeniedPage = () => {
  const intl = getIntl();
  return (
    <>
      <Helmet>
        <title>{intl.get('titles.403')}</title>
      </Helmet>
      <Result
        status="403"
        title="403"
        subTitle={intl.get('pages.403.text')}
        extra={
          <MicroAppLink isMaster to="/dashboard">
            <Button type="primary">
              <FormattedMessage id="pages.403.backhome" />
            </Button>
          </MicroAppLink>
        }
      />
    </>
  );
};

export default DeniedPage;
