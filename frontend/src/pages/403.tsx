import { getIntl } from '@/utils';
import { Button, Result } from 'antd';
import { FormattedMessage } from 'react-intl';
import { history, useModel } from 'umi';

const DeniedPage = () => {
  const intl = getIntl();
  const { initialState } = useModel('@@initialState');
  const content = (
    <Result
      status="403"
      title="403"
      subTitle={intl.get('pages.403.text')}
      extra={
        <Button
          type="primary"
          onClick={() =>
            history.push(initialState?.signin ? '/dashboard' : '/')
          }
        >
          <FormattedMessage id="pages.403.backhome" />
        </Button>
      }
    />
  );
  return <>{content}</>;
};

DeniedPage.title = 'titles.403';

export default DeniedPage;
