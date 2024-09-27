import { getIntl } from '@/utils';
import { DefaultFooter } from '@ant-design/pro-layout';
import styles from './style.less';

export default () => {
  const intl = getIntl();
  const currentYear = new Date().getFullYear();

  return (
    <DefaultFooter
      copyright={`${currentYear} ${intl.get('app.copyright.author')}`}
      links={[]}
      className={styles['footer']}
    />
  );
};
