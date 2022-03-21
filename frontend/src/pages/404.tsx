import { getIntl } from '@/utils';
import { Button, Result } from 'antd';
import { FormattedMessage } from 'react-intl';
import { history, useModel } from 'umi';

const NoFoundPage: { title: any; path: any; exact: any } = () => {
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
  return <>{content}</>;
};

NoFoundPage.title = 'titles.404';
NoFoundPage.path = undefined;
NoFoundPage.exact = undefined;

export default NoFoundPage;
