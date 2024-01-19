import { getIntl } from '@/utils';
import { Helmet, history, useModel } from '@umijs/max';
import { Button, Result } from 'antd';
import { FormattedMessage } from 'react-intl';

const DeniedPage = () => {
  const intl = getIntl();
  const { initialState } = useModel('@@initialState');
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
    </>
  );
};

export default DeniedPage;
