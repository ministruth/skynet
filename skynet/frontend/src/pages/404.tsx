import MainLayout from '@/components/layout';
import { getIntl } from '@/utils';
import { Helmet, history, useModel } from '@umijs/max';
import { Button, Result } from 'antd';
import { FormattedMessage } from 'react-intl';

const NoFoundPage = () => {
  const intl = getIntl();
  const { initialState } = useModel('@@initialState');
  const content = (
    <Result
      status="404"
      title="404"
      subTitle={intl.get('pages.404.text')}
      extra={
        <Button
          type="primary"
          onClick={() =>
            history.push(initialState?.signin ? '/dashboard' : '/')
          }
        >
          <FormattedMessage id="pages.404.backhome" />
        </Button>
      }
    />
  );
  return initialState?.signin ? (
    <MainLayout title="titles.404">{content}</MainLayout>
  ) : (
    <>
      <Helmet>
        <title>{intl.get('titles.404')}</title>
      </Helmet>
      {content}
    </>
  );
};

export default NoFoundPage;
