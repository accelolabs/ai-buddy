import { RouteInternTabs, RouteGoalCards, MainRouteBoard } from '@/widgets/trainee-route';

export function RoutePage() {
  return (
    <div className="min-h-full flex-1 bg-[#F8FAFC] p-4 sm:p-6 lg:p-8">
      <div className="mx-auto w-full max-w-[942px] pb-12">
        <RouteInternTabs />
        <RouteGoalCards />
        <MainRouteBoard />
      </div>
    </div>
  );
}

export default RoutePage;
