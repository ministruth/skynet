import { getIntl } from '@/utils';
import { Helmet, MicroAppLink } from '@umijs/max';
import { Button, Result } from 'antd';
import { FormattedMessage } from 'react-intl';

const NoFoundPage = () => {
  const intl = getIntl();
  return (
    <>
      <Helmet>
        <title>{intl.get('titles.404')}</title>
      </Helmet>
      <Result
        status="404"
        title="404"
        subTitle={intl.get('pages.404.text')}
        extra={
          <MicroAppLink isMaster to="/dashboard">
            <Button type="primary">
              <FormattedMessage id="pages.404.backhome" />
            </Button>
          </MicroAppLink>
        }
      />
    </>
  );
};

export default NoFoundPage;
