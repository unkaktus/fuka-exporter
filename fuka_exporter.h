#include <array>
#include <vector>

void copy_vector_to_array(double *arr, std::vector<double> v);

double *allocate_double(int n);

extern "C"
{
  typedef struct BinaryInfo
  {
    double mass1, mass2;             // Rest masses
    double position_x1, position_x2; // Compact object positions
  } BinaryInfo;

  typedef enum BinaryType
  {
    BNS,
    BBH,
    BHNS,
  } BinaryType;

  BinaryInfo read_binary_info(BinaryType binary_type, char *info_filename);

  typedef struct Grid
  {
    double *x, *y, *z;
    int n_points;
  } Grid;

  typedef struct Fields
  {
    double *alpha;
    double *beta_x, *beta_y, *beta_z;
    double *gamma_xx, *gamma_xy, *gamma_xz,
        *gamma_yy, *gamma_yz,
        *gamma_zz;
    double *K_xx, *K_xy, *K_xz,
        *K_yy, *K_yz,
        *K_zz;
    // Hydro variables
    double *rho;
    double *epsilon;
    double *pressure;
    double *v_x, *v_y, *v_z;
  } Fields;

  void free_fields(Fields *fields);

  typedef struct FUKAInterpolateRequest
  {
    BinaryType binary_type;
    char *info_filename;
    Grid *grid;
    double interpolation_offset;
    int interpolation_order;
    double relative_dr_spacing;
  } FUKAInterpolateRequest;

  Fields interpolate_FUKA_ID(FUKAInterpolateRequest *req);
  Fields allocate_fields(int n_points);
}

// Vacuum
enum vacuum_quantities
{
  ALPHA,
  BETAX,
  BETAY,
  BETAZ,
  GXX,
  GXY,
  GXZ,
  GYY,
  GYZ,
  GZZ,
  KXX,
  KXY,
  KXZ,
  KYY,
  KYZ,
  KZZ,
  N_VARIABLES_VACUUM
};
// Vacuum quantities extended with the matter quantities
enum matter_quantities
{
  RHO = N_VARIABLES_VACUUM,
  EPS,
  PRESS,
  VELX,
  VELY,
  VELZ,
  N_VARIABLES_MATTER
};

std::array<std::vector<double>, N_VARIABLES_MATTER> KadathExportBNS(int const npoints,
                                                                    double const *xx, double const *yy, double const *zz,
                                                                    char const *fn);

std::array<std::vector<double>, N_VARIABLES_VACUUM> KadathExportBBH(int const npoints,
                                                                    double const *xx, double const *yy, double const *zz,
                                                                    char const *fn,
                                                                    double const interpolation_offset, int const interp_order,
                                                                    double const delta_r_rel);

std::array<std::vector<double>, N_VARIABLES_MATTER> KadathExportBHNS(int const npoints,
                                                                     double const *xx, double const *yy, double const *zz,
                                                                     char const *fn,
                                                                     double const interpolation_offset, int const interp_order,
                                                                     double const delta_r_rel);